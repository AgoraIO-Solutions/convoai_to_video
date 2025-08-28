import requests
import json
import logging

# Configuration
API_ENDPOINT = "https://api.example.com/session/stop"  # Update with actual endpoint
# API_ENDPOINT = "http://localhost:8080/session/stop"  # For local testing
API_KEY = "YOUR_API_KEY"

# Example session token (in practice, this would come from session/start response)
EXAMPLE_SESSION_TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def test_session_stop_endpoint():
    """Test the session stop DELETE endpoint"""
    
    # Prepare test payload
    payload = {
        "session_token": EXAMPLE_SESSION_TOKEN
    }
    
    # API key in headers for security
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": API_KEY
    }
    
    logger.info(f"Testing endpoint: {API_ENDPOINT}")
    logger.info(f"Headers (API key masked): {dict(headers, **{'x-api-key': '***masked***'})}")
    logger.info(f"Payload: {json.dumps(payload, indent=2)}")
    
    try:
        # Send DELETE request
        response = requests.delete(
            API_ENDPOINT,
            headers=headers,
            json=payload,
            timeout=30
        )
        
        logger.info(f"Response status code: {response.status_code}")
        logger.info(f"Response headers: {dict(response.headers)}")
        
        # Parse response
        if response.headers.get('content-type', '').startswith('application/json'):
            response_data = response.json()
            logger.info(f"Response data: {json.dumps(response_data, indent=2)}")
        else:
            logger.info(f"Response text: {response.text}")
            response_data = {}
        
        # Verify successful response
        if response.status_code == 200:
            logger.info("✅ Request successful!")
            return verify_success_response(response_data)
        else:
            logger.error(f"❌ Request failed with status {response.status_code}")
            return verify_error_response(response_data, response.status_code)
            
    except requests.exceptions.ConnectionError as e:
        logger.error(f"❌ Connection error: {e}")
        logger.error("Make sure the API endpoint is running and accessible")
        return False
    except requests.exceptions.Timeout as e:
        logger.error(f"❌ Request timeout: {e}")
        return False
    except requests.exceptions.RequestException as e:
        logger.error(f"❌ Request error: {e}")
        return False
    except Exception as e:
        logger.error(f"❌ Unexpected error: {e}")
        return False


def verify_success_response(data):
    """Verify that success response contains expected fields"""
    logger.info("Verifying success response structure...")
    
    required_fields = ["status", "message"]
    missing_fields = []
    
    for field in required_fields:
        if field not in data:
            missing_fields.append(field)
        else:
            logger.info(f"✅ Found required field: {field}")
    
    if missing_fields:
        logger.error(f"❌ Missing required fields: {missing_fields}")
        return False
    
    # Verify status field
    status = data.get("status")
    if status != "success":
        logger.error(f"❌ Invalid status: {status}, expected 'success'")
        return False
    else:
        logger.info(f"✅ Valid status: {status}")
    
    # Verify message field is not empty
    message = data.get("message", "")
    if not message or len(message.strip()) == 0:
        logger.error("❌ message field is empty")
        return False
    else:
        logger.info(f"✅ Message present: {message}")
    
    # Check for unexpected fields (session_token should not be present)
    unexpected_fields = []
    for field in data:
        if field not in required_fields:
            unexpected_fields.append(field)
    
    if unexpected_fields:
        logger.warning(f"⚠️ Unexpected fields found: {unexpected_fields}")
        # This is a warning, not an error, so we don't fail the test
    
    logger.info("✅ All response validation checks passed!")
    return True


def verify_error_response(data, status_code):
    """Verify that error response contains expected fields"""
    logger.info(f"Verifying error response structure for status {status_code}...")
    
    if not data:
        logger.warning("No response data received for error")
        return False
    
    # Check for common error fields
    error_fields = ["error", "message"]
    found_fields = []
    
    for field in error_fields:
        if field in data:
            found_fields.append(field)
            logger.info(f"✅ Found error field: {field} = {data[field]}")
    
    if not found_fields:
        logger.warning("❌ No standard error fields found in response")
        return False
    
    logger.info("✅ Error response structure is valid")
    return True


def test_invalid_api_key():
    """Test the endpoint with an invalid API key"""
    logger.info("\n" + "="*50)
    logger.info("Testing with invalid API key...")
    
    payload = {
        "session_token": EXAMPLE_SESSION_TOKEN
    }
    
    # Invalid API key in headers
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": "INVALID_API_KEY"
    }
    
    try:
        response = requests.delete(API_ENDPOINT, headers=headers, json=payload, timeout=30)
        logger.info(f"Response status code: {response.status_code}")
        
        if response.status_code == 401:
            logger.info("✅ Correctly received 401 Unauthorized for invalid API key")
            return True
        else:
            logger.warning(f"⚠️ Expected 401 but got {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Error during invalid API key test: {e}")
        return False


def test_missing_api_key():
    """Test the endpoint with missing API key header"""
    logger.info("\n" + "="*50)
    logger.info("Testing with missing API key header...")
    
    payload = {
        "session_token": EXAMPLE_SESSION_TOKEN
    }
    
    # Headers without API key
    headers = {
        "accept": "application/json",
        "content-type": "application/json"
        # Missing x-api-key header
    }
    
    try:
        response = requests.delete(API_ENDPOINT, headers=headers, json=payload, timeout=30)
        logger.info(f"Response status code: {response.status_code}")
        
        if response.status_code in [401, 403]:
            logger.info(f"✅ Correctly received {response.status_code} for missing API key")
            return True
        else:
            logger.warning(f"⚠️ Expected 401 or 403 but got {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Error during missing API key test: {e}")
        return False


def test_missing_session_token():
    """Test the endpoint with missing session_token"""
    logger.info("\n" + "="*50)
    logger.info("Testing with missing session_token...")
    
    # Empty payload (missing session_token)
    payload = {}
    
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": API_KEY
    }
    
    try:
        response = requests.delete(API_ENDPOINT, headers=headers, json=payload, timeout=30)
        logger.info(f"Response status code: {response.status_code}")
        
        if response.status_code == 400:
            logger.info("✅ Correctly received 400 Bad Request for missing session_token")
            return True
        else:
            logger.warning(f"⚠️ Expected 400 but got {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Error during missing session_token test: {e}")
        return False


def test_invalid_session_token():
    """Test the endpoint with invalid session_token"""
    logger.info("\n" + "="*50)
    logger.info("Testing with invalid session_token...")
    
    payload = {
        "session_token": "invalid_token_that_does_not_exist"
    }
    
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": API_KEY
    }
    
    try:
        response = requests.delete(API_ENDPOINT, headers=headers, json=payload, timeout=30)
        logger.info(f"Response status code: {response.status_code}")
        
        if response.status_code == 404:
            logger.info("✅ Correctly received 404 Not Found for invalid session_token")
            return True
        else:
            logger.warning(f"⚠️ Expected 404 but got {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Error during invalid session_token test: {e}")
        return False


def main():
    """Run all tests"""
    logger.info("=" * 60)
    logger.info("SESSION STOP ENDPOINT TEST")
    logger.info("=" * 60)
    
    # Test 1: Valid request
    logger.info("\n" + "="*50)
    logger.info("Test 1: Valid stop session request")
    logger.info("="*50)
    
    success1 = test_session_stop_endpoint()
    
    # Test 2: Invalid API key
    logger.info("\n" + "="*50)
    logger.info("Test 2: Invalid API key")
    logger.info("="*50)
    
    success2 = test_invalid_api_key()
    
    # Test 3: Missing API key
    logger.info("\n" + "="*50)
    logger.info("Test 3: Missing API key header")
    logger.info("="*50)
    
    success3 = test_missing_api_key()
    
    # Test 4: Missing session token
    logger.info("\n" + "="*50)
    logger.info("Test 4: Missing session token")
    logger.info("="*50)
    
    success4 = test_missing_session_token()
    
    # Test 5: Invalid session token
    logger.info("\n" + "="*50)
    logger.info("Test 5: Invalid session token")
    logger.info("="*50)
    
    success5 = test_invalid_session_token()
    
    # Summary
    logger.info("\n" + "="*50)
    logger.info("TEST SUMMARY")
    logger.info("="*50)
    
    logger.info(f"Valid request test: {'✅ PASSED' if success1 else '❌ FAILED'}")
    logger.info(f"Invalid API key test: {'✅ PASSED' if success2 else '❌ FAILED'}")
    logger.info(f"Missing API key test: {'✅ PASSED' if success3 else '❌ FAILED'}")
    logger.info(f"Missing session token test: {'✅ PASSED' if success4 else '❌ FAILED'}")
    logger.info(f"Invalid session token test: {'✅ PASSED' if success5 else '❌ FAILED'}")
    
    total_passed = sum([success1, success2, success3, success4, success5])
    logger.info(f"\nOverall: {total_passed}/5 tests passed")
    
    if total_passed == 5:
        logger.info("🎉 All tests passed!")
    else:
        logger.info("⚠️ Some tests failed. Check the logs above for details.")


if __name__ == "__main__":
    main()