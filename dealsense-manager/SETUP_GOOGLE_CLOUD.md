# Google Cloud Setup Guide for DealSense

This guide walks you through setting up Google Cloud Platform services required for DealSense's document processing, embedding, and analysis features.

## Prerequisites

- A Google Cloud Platform account
- `gcloud` CLI installed (optional but recommended)
- Project billing enabled

## Step 1: Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click "Select a project" → "New Project"
3. Enter project name (e.g., "dealsense-production")
4. Note your **Project ID** (you'll need this later)

## Step 2: Enable Required APIs

Enable the following APIs in your project:

```bash
gcloud services enable storage.googleapis.com
gcloud services enable documentai.googleapis.com
gcloud services enable aiplatform.googleapis.com
gcloud services enable generativelanguage.googleapis.com
```

Or enable via Console:
1. Go to "APIs & Services" → "Enable APIs and Services"
2. Search and enable:
   - **Cloud Storage API**
   - **Document AI API**
   - **Vertex AI API**
   - **Generative Language API** (for Gemini)

## Step 3: Create a Service Account

1. Go to "IAM & Admin" → "Service Accounts"
2. Click "Create Service Account"
3. Name it `dealsense-service-account`
4. Grant the following roles:
   - **Storage Admin** (for GCS bucket access)
   - **Document AI User** (for Document AI processing)
   - **Vertex AI User** (for embeddings)
   - **AI Platform Developer** (for Vertex AI)

5. Click "Done"

## Step 4: Create and Download Service Account Key

1. Click on the service account you just created
2. Go to "Keys" tab
3. Click "Add Key" → "Create New Key"
4. Choose "JSON" format
5. Click "Create" - the key file will download automatically
6. **IMPORTANT**: Store this file securely! Never commit it to version control

## Step 5: Create Google Cloud Storage Bucket

```bash
# Create bucket (choose a globally unique name)
gsutil mb -p genai-exchange-475318 -c STANDARD -l us-central1 gs://dealsense-documents

# Set bucket permissions
gsutil iam ch serviceAccount:dealsense-service-account@genai-exchange-475318.iam.gserviceaccount.com:roles/storage.objectAdmin gs://dealsense-documents
```

Or via Console:
1. Go to "Cloud Storage" → "Buckets"
2. Click "Create Bucket"
3. Name: `dealsense-documents` (or your preferred name)
4. Location: `us-central1` (or your preferred region)
5. Storage class: Standard
6. Access control: Uniform
7. Click "Create"

## Step 6: Set Up Document AI Processor

1. Go to "Document AI" in GCP Console
2. Click "Create Processor"
3. Select processor type: **OCR Processor** (or **Form Parser** for better results)
4. Choose region: `us` or `eu`
5. Name it `dealsense-doc-processor`
6. Click "Create"
7. **Note the Processor ID** from the processor details page - 4e6939bf485e935d

## Step 7: Configure DealSense Backend

### Option A: Using Service Account Key File (Recommended for Development)

1. Place the downloaded JSON key file in a secure location (e.g., `/etc/dealsense/service-account-key.json`)

2. Set environment variable:
```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
```

3. Update `backend_v2/config.yaml`:
```yaml
google:
  api_key: "YOUR_GEMINI_API_KEY"
  project_id: "genai-exchange-475318"
  
  storage:
    bucket_name: "dealsense-documents"
    use_default_credentials: true
  
  document_ai:
    location: "us"  # or "eu"
    processor_id: "4e6939bf485e935d"
    use_default_credentials: true
  
  vertex_ai:
    location: "us-central1"
    embedding_model: "text-embedding-004"
    use_default_credentials: true
```

### Option B: Inline Credentials (Not Recommended for Production)

Update `config.yaml`:
```yaml
google:
  api_key: "YOUR_GEMINI_API_KEY"
  project_id: "genai-exchange-475318"
  
  storage:
    bucket_name: "dealsense-documents"
    use_default_credentials: false
    credentials_json: |
      {
        "type": "service_account",
        "project_id": "...",
        ... paste entire JSON key content ...
      }
```

## Step 8: Verify Setup

### Test Google Cloud Storage
```bash
# List buckets (should show your bucket)
gsutil ls

# Test upload
echo "test" > test.txt
gsutil cp test.txt gs://dealsense-documents/
gsutil rm gs://dealsense-documents/test.txt
rm test.txt
```

### Test Document AI
```bash
# Use gcloud to test (requires a sample PDF)
gcloud document-ai process \
  --processor-id=4e6939bf485e935d \
  --location=us \
  --input-file-path=sample.pdf \
  --output-file-path=output.json
```

### Test Vertex AI Embeddings
```bash
# Test via Python (requires google-cloud-aiplatform library)
python3 << EOF
from google.cloud import aiplatform

aiplatform.init(project="genai-exchange-475318", location="us-central1")

# Test embedding generation
from vertexai.language_models import TextEmbeddingModel
model = TextEmbeddingModel.from_pretrained("text-embedding-004")
embeddings = model.get_embeddings(["test text"])
print(f"Embedding dimension: {len(embeddings[0].values)}")
print("✓ Vertex AI embeddings working!")
EOF
```

## Step 9: Start DealSense Backend

```bash
cd dealsense-manager/backend_v2

# Set environment variable (if using service account key)
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"

# Run the backend
go run cmd/server/main.go
```

## Step 10: Verify Integration

1. Start the frontend: `cd ../frontend && npm run dev`
2. Create an agent in analyst mode
3. Navigate to the agent details page
4. Go to the "Documents" tab
5. Upload a PDF pitch deck
6. Wait for processing to complete
7. Go to "Chat" tab and ask questions about the document
8. Go to "Startup" tab and run analysis

## Troubleshooting

### "Permission denied" errors
- Verify service account has correct roles
- Check that GOOGLE_APPLICATION_CREDENTIALS points to valid key file
- Ensure APIs are enabled in your project

### Document processing fails
- Verify Document AI processor is created and active
- Check processor ID is correct in config.yaml
- Ensure document is under 50MB and is PDF/DOCX/PPTX

### Embedding generation fails
- Verify Vertex AI API is enabled
- Check location matches between config and API availability
- Ensure project has billing enabled

### Storage upload fails
- Verify bucket exists and name matches config
- Check service account has Storage Admin role
- Ensure bucket and service account are in same project

## Security Best Practices

1. **Never commit service account keys to version control**
   - Add `*service-account*.json` to `.gitignore`
   - Use environment variables or secret managers

2. **Rotate service account keys regularly**
   - Create new keys every 90 days
   - Delete old keys after rotation

3. **Use least privilege principle**
   - Only grant necessary roles
   - Create separate service accounts for different environments

4. **Enable audit logging**
   - Monitor Document AI usage
   - Track storage access patterns
   - Review embedding API calls

5. **Set up budget alerts**
   - Document AI: ~$1.50 per 1000 pages
   - Vertex AI embeddings: ~$0.025 per 1000 embeddings
   - GCS storage: ~$0.02 per GB/month

## Cost Estimates

For a typical startup analysis workflow:

- **Document Processing**: $1.50 per 1000 pages (Document AI)
- **Embeddings**: $0.025 per 1000 chunks (~$0.05 per typical pitch deck)
- **Storage**: Negligible (~$0.001 per document per month)
- **Gemini API**: Varies by model and usage

**Example**: 100 pitch decks per month
- Processing: $15
- Embeddings: $5
- Storage: $0.10
- **Total: ~$20/month** (excluding Gemini API calls)

## Support

For issues:
1. Check GCP status page: https://status.cloud.google.com/
2. Review Cloud Logging in GCP Console
3. Check DealSense logs: `backend_v2/logs/dealsense.log`
4. Refer to official docs:
   - [Document AI](https://cloud.google.com/document-ai/docs)
   - [Vertex AI](https://cloud.google.com/vertex-ai/docs)
   - [Cloud Storage](https://cloud.google.com/storage/docs)

