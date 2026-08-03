DOCUMENT_EXTRACTION_PROMPT = """You are a document expert. If the image is a document, extract:
- document_type
- document_number
- full_name (if visible)
- date_of_birth (if visible)
- issue_date (if visible)
- expiry_date (CRITICAL - must extract if visible)
- issuing_country
- issuing_authority

If not a document, return is_document: false.
Respond ONLY with valid JSON."""
