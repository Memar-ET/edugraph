package repository

import (
	"context"
	"fmt"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	apperrors "github.com/edugraph-ai/edugraph/pkg/errors"
)

// GetSubjectGraph queries Neo4j for a subject's full knowledge-graph
// subtree (Subject -> Units -> Topics -> Subtopics [-> CLOs]) plus every
// HAS_PREREQUISITE edge touching one of those topics -- including the
// topic at the *other* end when it belongs to a different subject
// (cross-grade prerequisites, e.g. a BIO-G9 topic requiring a BIO-G7
// one), rendered as a minimal "external" node rather than silently
// dropped or expanded into its own subject's full subtree.
//
// Every promoted topic (top-level or subtopic) has a direct
// (:Unit)-[:HAS_TOPIC]->(:Topic) edge (see syncTopicToNeo4j), so a single
// unit->topic match already reaches subtopics too -- no separate
// HAS_SUBTOPIC traversal is needed to find them, only to draw their
// nesting edge.
func (r *Repository) GetSubjectGraph(ctx context.Context, subjectCode string, includeClos bool) (*dto.SubjectGraph, error) {
	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)

	nodes := make(map[string]dto.GraphNode)
	var edges []dto.GraphEdge

	subjRows, err := session.Run(ctx,
		`MATCH (s:Subject {code: $code}) RETURN s.code AS code, s.gradeLevel AS gradeLevel`,
		map[string]any{"code": subjectCode},
	)
	if err != nil {
		return nil, fmt.Errorf("query subject: %w", err)
	}
	subjRecord, err := subjRows.Single(ctx)
	if err != nil {
		return nil, apperrors.NotFound(fmt.Sprintf("subject %q not found in the knowledge graph", subjectCode))
	}
	gradeLevel, _ := subjRecord.Get("gradeLevel")
	nodes[subjectCode] = dto.GraphNode{ID: subjectCode, Label: subjectCode, Type: "subject", GradeLevel: asIntPtr(gradeLevel), SubjectCode: subjectCode}

	unitRecords, err := runAndCollect(ctx, session, `
		MATCH (s:Subject {code: $code})-[:HAS_UNIT]->(u:Unit)
		RETURN u.id AS id, u.number AS number, u.titleEn AS title
	`, map[string]any{"code": subjectCode})
	if err != nil {
		return nil, fmt.Errorf("query units: %w", err)
	}
	for _, rec := range unitRecords {
		id, _ := rec.Get("id")
		number, _ := rec.Get("number")
		title, _ := rec.Get("title")
		idStr, _ := id.(string)
		nodes[idStr] = dto.GraphNode{ID: idStr, Label: fmt.Sprintf("Unit %v: %v", number, title), Type: "unit", SubjectCode: subjectCode}
		edges = append(edges, dto.GraphEdge{Source: subjectCode, Target: idStr, Type: "HAS_UNIT"})
	}

	topicRecords, err := runAndCollect(ctx, session, `
		MATCH (s:Subject {code: $code})-[:HAS_UNIT]->(u:Unit)-[:HAS_TOPIC]->(t:Topic)
		OPTIONAL MATCH (parent:Topic)-[:HAS_SUBTOPIC]->(t)
		RETURN u.id AS unitId, t.id AS id, t.titleEn AS title, t.gradeLevel AS gradeLevel, parent.id AS parentId
	`, map[string]any{"code": subjectCode})
	if err != nil {
		return nil, fmt.Errorf("query topics: %w", err)
	}
	for _, rec := range topicRecords {
		unitID, _ := rec.Get("unitId")
		id, _ := rec.Get("id")
		title, _ := rec.Get("title")
		topicGrade, _ := rec.Get("gradeLevel")
		parentID, _ := rec.Get("parentId")

		idStr, _ := id.(string)
		isSubtopic := parentID != nil
		nodes[idStr] = dto.GraphNode{
			ID: idStr, Label: fmt.Sprintf("%v", title), Type: "topic",
			GradeLevel: asIntPtr(topicGrade), IsSubtopic: isSubtopic, SubjectCode: subjectCode,
		}
		if isSubtopic {
			parentIDStr, _ := parentID.(string)
			edges = append(edges, dto.GraphEdge{Source: parentIDStr, Target: idStr, Type: "HAS_SUBTOPIC"})
		} else {
			unitIDStr, _ := unitID.(string)
			edges = append(edges, dto.GraphEdge{Source: unitIDStr, Target: idStr, Type: "HAS_TOPIC"})
		}
	}

	if includeClos {
		cloRecords, err := runAndCollect(ctx, session, `
			MATCH (s:Subject {code: $code})-[:HAS_UNIT]->(:Unit)-[:HAS_TOPIC]->(t:Topic)-[:HAS_CLO]->(c:CLO)
			RETURN DISTINCT t.id AS topicId, c.code AS code
		`, map[string]any{"code": subjectCode})
		if err != nil {
			return nil, fmt.Errorf("query clos: %w", err)
		}
		for _, rec := range cloRecords {
			topicID, _ := rec.Get("topicId")
			code, _ := rec.Get("code")
			codeStr, _ := code.(string)
			nodes[codeStr] = dto.GraphNode{ID: codeStr, Label: codeStr, Type: "clo", SubjectCode: subjectCode}
			topicIDStr, _ := topicID.(string)
			edges = append(edges, dto.GraphEdge{Source: topicIDStr, Target: codeStr, Type: "HAS_CLO"})
		}
	}

	// Outgoing: this subject's topics -> whatever they require (possibly in another subject/grade).
	prereqOutRecords, err := runAndCollect(ctx, session, `
		MATCH (s:Subject {code: $code})-[:HAS_UNIT]->(:Unit)-[:HAS_TOPIC]->(t:Topic)
		MATCH (t)-[:HAS_PREREQUISITE]->(p:Topic)
		RETURN t.id AS fromId, p.id AS toId, p.titleEn AS toTitle, p.gradeLevel AS toGrade, p.subjectCode AS toSubjectCode
	`, map[string]any{"code": subjectCode})
	if err != nil {
		return nil, fmt.Errorf("query outgoing prerequisites: %w", err)
	}
	for _, rec := range prereqOutRecords {
		fromID, _ := rec.Get("fromId")
		toID, _ := rec.Get("toId")
		toTitle, _ := rec.Get("toTitle")
		toGrade, _ := rec.Get("toGrade")
		toSubjectCode, _ := rec.Get("toSubjectCode")
		toIDStr, _ := toID.(string)
		if _, exists := nodes[toIDStr]; !exists {
			toSubj, _ := toSubjectCode.(string)
			nodes[toIDStr] = dto.GraphNode{
				ID: toIDStr, Label: fmt.Sprintf("%v", toTitle), Type: "topic",
				GradeLevel: asIntPtr(toGrade), External: toSubj != subjectCode, SubjectCode: toSubj,
			}
		}
		fromIDStr, _ := fromID.(string)
		edges = append(edges, dto.GraphEdge{Source: fromIDStr, Target: toIDStr, Type: "HAS_PREREQUISITE"})
	}

	// Incoming: whatever requires one of this subject's topics (possibly from another subject/grade).
	prereqInRecords, err := runAndCollect(ctx, session, `
		MATCH (s:Subject {code: $code})-[:HAS_UNIT]->(:Unit)-[:HAS_TOPIC]->(t:Topic)
		MATCH (d:Topic)-[:HAS_PREREQUISITE]->(t)
		RETURN d.id AS fromId, t.id AS toId, d.titleEn AS fromTitle, d.gradeLevel AS fromGrade, d.subjectCode AS fromSubjectCode
	`, map[string]any{"code": subjectCode})
	if err != nil {
		return nil, fmt.Errorf("query incoming prerequisites: %w", err)
	}
	for _, rec := range prereqInRecords {
		fromID, _ := rec.Get("fromId")
		toID, _ := rec.Get("toId")
		fromTitle, _ := rec.Get("fromTitle")
		fromGrade, _ := rec.Get("fromGrade")
		fromSubjectCode, _ := rec.Get("fromSubjectCode")
		fromIDStr, _ := fromID.(string)
		if _, exists := nodes[fromIDStr]; !exists {
			fromSubj, _ := fromSubjectCode.(string)
			nodes[fromIDStr] = dto.GraphNode{
				ID: fromIDStr, Label: fmt.Sprintf("%v", fromTitle), Type: "topic",
				GradeLevel: asIntPtr(fromGrade), External: fromSubj != subjectCode, SubjectCode: fromSubj,
			}
		}
		toIDStr, _ := toID.(string)
		edges = append(edges, dto.GraphEdge{Source: fromIDStr, Target: toIDStr, Type: "HAS_PREREQUISITE"})
	}

	out := &dto.SubjectGraph{Nodes: make([]dto.GraphNode, 0, len(nodes)), Edges: edges}
	for _, n := range nodes {
		out.Nodes = append(out.Nodes, n)
	}
	return out, nil
}

func runAndCollect(ctx context.Context, session neo4jdriver.SessionWithContext, cypher string, params map[string]any) ([]*neo4jdriver.Record, error) {
	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, err
	}
	return result.Collect(ctx)
}

func asIntPtr(v any) *int {
	switch t := v.(type) {
	case int64:
		n := int(t)
		return &n
	case int:
		return &t
	default:
		return nil
	}
}
