// (:CareerPath)-[:REQUIRES_SUBJECT]->(:Subject) models which subjects feed a career.
// (:Student)-[:MATCHED_TO {score}]->(:CareerPath) mirrors backend.career_matches
// so the career engine can traverse subject -> career recommendations in-graph.

CREATE INDEX career_title_lookup IF NOT EXISTS
FOR (c:CareerPath) ON (c.title);
