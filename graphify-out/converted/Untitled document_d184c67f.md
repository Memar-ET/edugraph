<!-- converted from Untitled document.docx -->

IMPROVISED IMPLEMENTATION OF PHASE (considering not using s3 bucket for now , instead saving it and processing it locally)

This is a brilliant architectural decision. Building an abstraction layer (often called the Strategy or Adapter pattern) right now means you can develop and test the entire system locally without paying for AWS, and when you are ready for production, you just flip a configuration switch. No code rewriting required. Here is the updated 4-step implementation plan, featuring the Dual-Storage System.
The Secret Sauce:

The Storage Interface
Before we look at the steps, here is how we make the swap effortless. In both Go and Python, we will define a simple "Interface" (a contract).

// Go Backend: pkg/storage/interface.go
type StorageProvider interface {
Upload(ctx context.Context, fileName string, file io.Reader) (string, error)
Download(ctx context.Context, fileRef string) (io.ReadCloser, error)
}
In Development: We inject a LocalStorageProvider (saves to your hard drive or a Postgres BYTEA column).
In Production: We inject an S3StorageProvider (talks to AWS)
The rest of your code never knows or cares which one is being used.

Step 1: The Upload (Go API + Dual Storage)
The Goal: Securely accept the PDF and store it using the active storage provider without crashing the server.
The curriculum officer uploads a PDF via the frontend.
The Go Backend receives the request. It checks the JWT to ensure the user has the curriculum_officer or ministry role.
The Fork in the Road: Go passes the file to the StorageProvider.
Dev Mode (Active Now): The LocalStorageProvider saves the file to a local directory (e.g., ./data/uploads/) or directly into a Postgres BYTEA column. It returns a local file path or reference ID.
Prod Mode (Future): The S3StorageProvider generates a Presigned URL, and the frontend uploads directly to the edugraph-curriculum-docs S3 bucket.
Go creates a row in the curriculum.upload_jobs table with the status 'pending' and stores the file reference (local path or S3 key).
Go pushes a message to Redis saying: "Hey, there is a new PDF at [reference] that needs parsing!"

Step 2: The Brain Work (Python AI Service + Dual Storage)
The Goal: Read the PDF like a human would, identifying the Units, Chapters, and Topics.
The Python AI Service (running Celery workers) picks up the job from Redis.
The Fork in the Road: The Python service uses its own StorageProvider (configured via an environment variable like STORAGE_BACKEND=local).
Dev Mode (Active Now): It reads the file from the local shared directory or fetches the BYTEA data directly from Postgres into memory.
Prod Mode (Future): It uses the AWS SDK to download the PDF from S3 into its temporary memory.
It uses PyMuPDF (fitz) to read the document using the smart two-step strategy:
Strategy A (The TOC): Checks for a Table of Contents to use as the exact skeleton.
Strategy B (Font Heuristics): If no TOC, looks at font sizes (e.g., 18pt Bold = Unit, 14pt Bold = Topic).
It extracts the text under each heading to pull out "Key Concepts" and "Learning Outcomes".
It packages all of this into a structured JSON tree, saves it to the parsed_structure column in Postgres, and updates the job status to 'parsed'.

Step 3: The Human Review (Frontend + Go API)
The Goal: AI is smart, but not perfect. A human must verify the structure before it becomes the "source of truth" for the whole country.
The frontend fetches the parsed JSON tree via GET /api/v1/curriculum/jobs/{jobId}.
The Fork in the Road (Viewing the PDF): If the officer wants to view the original PDF alongside the parsed tree:
Dev Mode: The frontend requests the file via a Go proxy endpoint (GET /api/v1/storage/files/{jobId}), and Go streams the local file/Postgres bytea to the browser.
Prod Mode: The frontend requests a presigned download URL from Go, and streams it directly from S3.
The curriculum officer sees a visual tree: "Unit 1: Mechanics -> Topic 1.1: Kinematics".
If the AI missed a subtopic or merged two chapters, the officer can edit it directly in the UI.
Once satisfied, the officer clicks "Approve".
The frontend sends a POST /api/v1/curriculum/jobs/{jobId}/approve request to the Go Backend.

Step 4: The Finalization (Go API → PostgreSQL → Neo4j)
The Goal: Lock in the approved data and activate the Knowledge Graph. (Note: This step is identical in both Dev and Prod, as it deals with data, not files).
The Go Backend receives the approved JSON.
It writes the final, verified data into the relational tables: curriculum.subjects, curriculum.units, and curriculum.topics.
It updates the upload_jobs status to 'approved'.
The Magic Moment: The Go backend (or the sync worker) takes these new PostgreSQL rows and writes them into Neo4j as Nodes (:Subject, :Unit, :Topic) and Relationships (:HAS_UNIT, :HAS_TOPIC).
The curriculum is now officially "in the brain" and ready for Phase 2 (Exam Validation)!
