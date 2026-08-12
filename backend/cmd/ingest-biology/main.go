// Command ingest-biology is a one-off importer for the Ethiopian MoE
// Biology G7-G12 unified curriculum document. It's not part of the
// normal AI-parsing pipeline (that pipeline is built for messy raw PDFs;
// this source document is already ID-tagged and pre-structured for KG
// ingestion, so a dedicated Python parser -- see
// scripts/parse_biology.py / transform_for_ingest.py -- extracts it
// deterministically into a JSON shape this command unmarshals directly
// into the real dto.ParsedStructurePayload).
//
// What it does, per grade (subject code "BIO-G7".."BIO-G12"):
//  1. Stores the source file via the normal StorageProvider and creates a
//     curriculum.upload_jobs row exactly as a real upload would (status
//     'parsed', parsed_structure set), so it's fully visible in the
//     Curriculum Officer dashboard and job-review screen afterward.
//  2. Calls the real Repository.ApproveAndPromote in-process -- same
//     promotion/validation/Neo4j-mirroring code path a human clicking
//     "Approve" hits, not a shortcut around it.
//
// After all six grades are promoted, it bulk-inserts the ~90 explicit
// prerequisite statements from the document's Part II via the real
// Service.AddTopicPrerequisite (cycle-checked, Neo4j-synced), resolving
// unit-level statements to an anchor topic per the agreed rule: first
// topic of the dependent unit, last topic of the prerequisite unit.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"strings"

	"github.com/google/uuid"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
	"github.com/edugraph-ai/edugraph/internal/curriculum/repository"
	"github.com/edugraph-ai/edugraph/internal/curriculum/service"
	"github.com/edugraph-ai/edugraph/pkg/config"
	dbneo4j "github.com/edugraph-ai/edugraph/pkg/database/neo4j"
	"github.com/edugraph-ai/edugraph/pkg/database/postgres"
	dbredis "github.com/edugraph-ai/edugraph/pkg/database/redis"
	"github.com/edugraph-ai/edugraph/pkg/storage"
)

type ingestFile struct {
	Jobs           []ingestJob         `json:"jobs"`
	Prerequisites  []prereqEdge        `json:"prerequisites"`
	UnitTopicOrder map[string][]string `json:"unitTopicOrder"`
}

type ingestJob struct {
	SubjectCode  string           `json:"subjectCode"`
	GradeLevel   int              `json:"gradeLevel"`
	AcademicYear string           `json:"academicYear"`
	Units        []dto.ParsedUnit `json:"units"`
}

type prereqEdge struct {
	Target            string `json:"target"`
	TargetGranularity string `json:"targetGranularity"`
	Prereq            string `json:"prereq"`
	PrereqGranularity string `json:"prereqGranularity"`
	Rationale         string `json:"rationale"`
}

func main() {
	jsonPath := flag.String("json", "", "path to the ingest-ready JSON produced by transform_for_ingest.py")
	sourceFile := flag.String("source-file", "", "path to the original source document (stored per job, same as a normal upload)")
	officerEmail := flag.String("officer-email", "curriculum.officer@edugraph.et", "uploaded_by user for the created upload_jobs rows")
	wipeNeo4j := flag.Bool("wipe-neo4j", false, "DETACH DELETE every node in Neo4j before ingesting (irreversible -- confirm with the user first)")
	flag.Parse()

	if *jsonPath == "" || *sourceFile == "" {
		log.Fatal("usage: ingest-biology -json biology_ingest.json -source-file <path to .docx> [-wipe-neo4j]")
	}

	ctx := context.Background()
	cfg := config.Load()

	pool, err := postgres.NewPool(cfg.Postgres)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	neo4jDriver, err := dbneo4j.NewDriver(cfg.Neo4j)
	if err != nil {
		log.Fatalf("connect neo4j: %v", err)
	}
	defer neo4jDriver.Close(ctx)

	redisClient, err := dbredis.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	if *wipeNeo4j {
		log.Println("wiping Neo4j (MATCH (n) DETACH DELETE n)...")
		session := neo4jDriver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
		if _, err := session.Run(ctx, "MATCH (n) DETACH DELETE n", nil); err != nil {
			session.Close(ctx)
			log.Fatalf("wipe neo4j: %v", err)
		}
		session.Close(ctx)
		log.Println("neo4j wiped.")
	}

	raw, err := os.ReadFile(*jsonPath)
	if err != nil {
		log.Fatalf("read json: %v", err)
	}
	var f ingestFile
	if err := json.Unmarshal(raw, &f); err != nil {
		log.Fatalf("parse json: %v", err)
	}

	fileBytes, err := os.ReadFile(*sourceFile)
	if err != nil {
		log.Fatalf("read source file: %v", err)
	}
	fileName := *sourceFile
	if idx := strings.LastIndexAny(fileName, `/\`); idx >= 0 {
		fileName = fileName[idx+1:]
	}

	var officerID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, *officerEmail).Scan(&officerID); err != nil {
		log.Fatalf("look up officer %q: %v", *officerEmail, err)
	}

	repo := repository.New(pool, neo4jDriver)
	storageProvider := storage.NewPostgresStorage(pool)

	topicIDByCode := make(map[string]uuid.UUID)

	for _, job := range f.Jobs {
		log.Printf("promoting %s (grade %d, %d units)...", job.SubjectCode, job.GradeLevel, len(job.Units))

		fileRef, err := storageProvider.Upload(ctx, fileName,
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			bytes.NewReader(fileBytes))
		if err != nil {
			log.Fatalf("%s: store file: %v", job.SubjectCode, err)
		}

		structure := dto.ParsedStructurePayload{Units: job.Units}
		structureJSON, err := json.Marshal(structure)
		if err != nil {
			log.Fatalf("%s: marshal structure: %v", job.SubjectCode, err)
		}

		var jobID uuid.UUID
		err = pool.QueryRow(ctx, `
			INSERT INTO curriculum.upload_jobs
				(uploaded_by, subject_code, grade_level, academic_year, file_s3_key, file_name, file_size_bytes, status, parsed_structure)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'parsed', $8::jsonb)
			RETURNING id
		`, officerID, job.SubjectCode, job.GradeLevel, job.AcademicYear, fileRef, fileName, len(fileBytes), structureJSON).Scan(&jobID)
		if err != nil {
			log.Fatalf("%s: create upload job: %v", job.SubjectCode, err)
		}

		resp, result, err := repo.ApproveAndPromote(ctx, jobID, officerID, structure, structureJSON)
		if err != nil {
			log.Fatalf("%s: approve and promote: %v", job.SubjectCode, err)
		}
		log.Printf("  -> %d units, %d topics, %d CLOs promoted (graphSynced=%v%s)",
			resp.UnitsPromoted, resp.TopicsPromoted, resp.ClosPromoted, resp.GraphSynced, graphErrSuffix(resp.GraphSyncError))

		maps.Copy(topicIDByCode, result.TopicIDByCode)

		// Feature 1.1: same best-effort embedding-queue push Service.Approve
		// does for a normal HTTP approval -- duplicated here (not called
		// through the service layer) because this command needs
		// PromotionResult.TopicIDByCode, which Service.Approve doesn't
		// expose through its HTTP-facing dto.ApproveResponse return type.
		for _, target := range result.EmbeddingTargets {
			payload := map[string]string{"kind": target.Kind}
			if target.Kind == "topic" {
				payload["id"] = target.TopicID.String()
			} else {
				payload["code"] = target.CloCode
			}
			b, err := json.Marshal(payload)
			if err != nil {
				continue
			}
			if err := redisClient.LPush(ctx, "queue:embedding:generate", b).Err(); err != nil {
				log.Printf("  embedding queue push failed for %s: %v", target.Kind, err)
			}
		}
	}

	log.Printf("promoted %d grades, %d topics resolvable by external code", len(f.Jobs), len(topicIDByCode))

	// ---- Prerequisites (Part II) ----
	svc := service.New(repo, storageProvider, redisClient)
	weight := 1.0
	synced, failed := 0, 0

	resolve := func(code, granularity, side string) (uuid.UUID, error) {
		switch granularity {
		case "topic":
			id, ok := topicIDByCode[code]
			if !ok {
				return uuid.Nil, fmt.Errorf("topic %q not found among promoted topics", code)
			}
			return id, nil
		case "unit":
			order, ok := f.UnitTopicOrder[code]
			if !ok || len(order) == 0 {
				return uuid.Nil, fmt.Errorf("unit %q has no topics to anchor to", code)
			}
			anchor := order[0] // dependent side -> first topic of the unit
			if side == "prereq" {
				anchor = order[len(order)-1] // prerequisite side -> last topic of the unit
			}
			id, ok := topicIDByCode[anchor]
			if !ok {
				return uuid.Nil, fmt.Errorf("anchor topic %q (unit %q) not found", anchor, code)
			}
			return id, nil
		default:
			return uuid.Nil, fmt.Errorf("unsupported granularity %q", granularity)
		}
	}

	for _, e := range f.Prerequisites {
		targetID, err := resolve(e.Target, e.TargetGranularity, "target")
		if err != nil {
			log.Printf("skip prereq %s REQUIRES %s: %v", e.Target, e.Prereq, err)
			failed++
			continue
		}
		prereqID, err := resolve(e.Prereq, e.PrereqGranularity, "prereq")
		if err != nil {
			log.Printf("skip prereq %s REQUIRES %s: %v", e.Target, e.Prereq, err)
			failed++
			continue
		}

		resp, err := svc.AddTopicPrerequisite(ctx, officerID, targetID, dto.AddPrerequisiteRequest{
			PrerequisiteTopicID: prereqID.String(),
			Weight:              &weight,
			InferMethod:         "moe_document",
		})
		if err != nil {
			log.Printf("prereq %s REQUIRES %s failed: %v", e.Target, e.Prereq, err)
			failed++
			continue
		}
		if !resp.GraphSynced {
			log.Printf("prereq %s REQUIRES %s: postgres ok, neo4j sync failed: %s", e.Target, e.Prereq, resp.GraphError)
		}
		synced++
	}

	log.Printf("prerequisites: %d synced, %d failed/skipped (of %d parsed)", synced, failed, len(f.Prerequisites))
}

func graphErrSuffix(errMsg string) string {
	if errMsg == "" {
		return ""
	}
	return fmt.Sprintf(", graphError=%q", errMsg)
}
