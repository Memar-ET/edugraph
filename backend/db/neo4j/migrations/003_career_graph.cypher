// 003_career_graph.cypher
// Career recommendation graph schema
// Mirrors PostgreSQL careers.* schema

// =====================================================
// 1. CAREER NODE CONSTRAINTS (already created in 002)
// =====================================================
// Career constraint already exists from 002:
// CREATE CONSTRAINT career_id_unique FOR (c:Career) REQUIRE c.id IS UNIQUE

// =====================================================
// 2. CAREER INDEXES
// =====================================================

// Career title search
CREATE INDEX career_title_lookup IF NOT EXISTS
FOR (c:Career) ON (c.title);

// Career sector + education level filtering
CREATE INDEX career_sector_edu_lookup IF NOT EXISTS
FOR (c:Career) ON (c.sector, c.minEduLevel);

// =====================================================
// 3. CAREER RELATIONSHIPS (Documentation)
// =====================================================

// Career requires topic
// (c:Career)-[:REQUIRES {importance: 'critical'|'important'|'helpful'}]->(t:Topic)

// Career requires subject (alternative model)
// (c:Career)-[:REQUIRES_SUBJECT]->(s:Subject)

// Student matched to career
// (s:Student)-[:MATCHED_TO {score, generatedAt}]->(c:Career)

// =====================================================
// 4. EXAMPLE QUERIES (for reference)
// =====================================================

// Career readiness scoring:
/*
MATCH (career:Career)-[:REQUIRES]->(required:Topic)
OPTIONAL MATCH (s:Student {id: $studentId})-[:MASTERED]->(required)
WITH career, COUNT(required) AS total, COUNT(s) AS mastered
RETURN career.name,
       ROUND(mastered * 100.0 / total) AS readinessPct
ORDER BY readinessPct DESC
LIMIT 5
*/

// Find careers by sector:
/*
MATCH (c:Career {sector: 'Engineering'})
RETURN c.name, c.minEduLevel
ORDER BY c.name
*/