import requests
import json
import logging

# Configuration for local testing
API_ENDPOINT_START = "http://localhost:8764/session/start"  # For creating test session
API_ENDPOINT_STOP = "http://localhost:8764/session/stop"   # Points to mock server
# API_ENDPOINT_STOP = "https://api.example.com/session/stop"  # For production testing
API_KEY = "test-api-key-123"  # Matches mock server default

# These will be populated by creating a real session first
test_session_id = None
test_session_token = None

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def create_test_session():
    """Create a test session to use for stop testing"""
    global test_session_id, test_session_token
    
    logger.info("Creating test session for stop endpoint testing...")
    
    payload = {
        "avatar_id": "test_avatar_for_stop_test",
        "quality": "high",
        "version": "v1",
        "video_encoding": "H264",
        "activity_idle_timeout": 120,
        "agora_settings": {
            "app_id": "test_app_id",
            "token": "test_token",
            "channel": "test_channel",
            "uid": "123",
            "enable_string_uid": False
        }
    }
    
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": API_KEY
    }
    
    try:
        response = requests.post(API_ENDPOINT_START, headers=headers, json=payload, timeout=30)
        
        if response.status_code == 200:
            data = response.json()
            if not isinstance(data, dict):
                logger.error("❌ Session creation response must be a JSON object")
                return False
            test_session_id = data.get("session_id")
            test_session_token = data.get("session_token")
            if not all(
                isinstance(value, str) and value.strip()
                for value in (test_session_id, test_session_token)
            ):
                logger.error("❌ Session creation response needs a non-empty session_id and session_token")
                return False
            
            logger.info(f"✅ Test session created successfully!")
            logger.info(f"Session ID: {test_session_id}")
            logger.info(f"Session Token: {test_session_token}")
            return True
        else:
            logger.error(f"❌ Failed to create test session: {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Error creating test session: {e}")
        return False


def test_session_stop_endpoint():
    """Test the session stop DELETE endpoint"""
    
    # First create a test session
    if not create_test_session():
        logger.error("❌ Cannot proceed with stop test - failed to create test session")
        return False
    
    # Prepare test payload with real session data
    payload = {
        "session_id": test_session_id,
        "session_token": test_session_token
    }
    
    # API key in headers for security
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": API_KEY
    }
    
    logger.info(f"Testing endpoint: {API_ENDPOINT_STOP}")
    logger.info(f"Headers (API key masked): {dict(headers, **{'x-api-key': '***masked***'})}")
    logger.info(f"Payload: {json.dumps(payload, indent=2)}")
    
    try:
        # Send DELETE request
        response = requests.delete(
            API_ENDPOINT_STOP,
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
            return False
            
    except requests.exceptions.ConnectionError as e:
        logger.error(f"❌ Connection error: {e}")
        logger.error("Make sure the mock server (session_test_receiver.py) is running on port 8764")
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

    if not isinstance(data, dict):
        logger.error("❌ Success response must be a JSON object")
        return False
    
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
    if not isinstance(message, str) or not message.strip():
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
    
    # Use hardcoded values for error testing (don't need real session)
    payload = {
        "session_id": "test_session_id",
        "session_token": "test_session_token"
    }
    
    # Invalid API key in headers
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": "INVALID_API_KEY"
    }
    
    try:
        response = requests.delete(API_ENDPOINT_STOP, headers=headers, json=payload, timeout=30)
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
    
    # Use hardcoded values for error testing (don't need real session)
    payload = {
        "session_id": "test_session_id",
        "session_token": "test_session_token"
    }
    
    # Headers without API key
    headers = {
        "accept": "application/json",
        "content-type": "application/json"
        # Missing x-api-key header
    }
    
    try:
        response = requests.delete(API_ENDPOINT_STOP, headers=headers, json=payload, timeout=30)
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


def test_missing_session_id():
    """Test the endpoint with missing session_id"""
    logger.info("\n" + "="*50)
    logger.info("Testing with missing session_id...")
    
    # Missing session_id field
    payload = {
        "session_token": "test_session_token"
    }
    
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": API_KEY
    }
    
    try:
        response = requests.delete(API_ENDPOINT_STOP, headers=headers, json=payload, timeout=30)
        logger.info(f"Response status code: {response.status_code}")
        
        if response.status_code == 400:
            logger.info("✅ Correctly received 400 Bad Request for missing session_id")
            return True
        else:
            logger.warning(f"⚠️ Expected 400 but got {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Error during missing session_id test: {e}")
        return False


def test_missing_session_token():
    """Test the endpoint with missing session_token"""
    logger.info("\n" + "="*50)
    logger.info("Testing with missing session_token...")
    
    # Missing session_token field
    payload = {
        "session_id": "test_session_id"
    }
    
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": API_KEY
    }
    
    try:
        response = requests.delete(API_ENDPOINT_STOP, headers=headers, json=payload, timeout=30)
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


def test_invalid_session_id():
    """Test the endpoint with invalid session_id"""
    logger.info("\n" + "="*50)
    logger.info("Testing with invalid session_id...")
    
    payload = {
        "session_id": "invalid_session_id_that_does_not_exist",
        "session_token": "test_session_token"
    }
    
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": API_KEY
    }
    
    try:
        response = requests.delete(API_ENDPOINT_STOP, headers=headers, json=payload, timeout=30)
        logger.info(f"Response status code: {response.status_code}")
        
        if response.status_code == 404:
            logger.info("✅ Correctly received 404 Not Found for invalid session_id")
            return True
        else:
            logger.warning(f"⚠️ Expected 404 but got {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Error during invalid session_id test: {e}")
        return False


def main():
    """Run all tests"""
    logger.info("=" * 60)
    logger.info("SESSION STOP ENDPOINT TEST (LOCAL)")
    logger.info("=" * 60)
    logger.info("NOTE: Make sure session_test_receiver.py is running on port 8764")
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
    
    # Test 4: Missing session_id
    logger.info("\n" + "="*50)
    logger.info("Test 4: Missing session_id")
    logger.info("="*50)
    
    success4 = test_missing_session_id()
    
    # Test 5: Missing session token
    logger.info("\n" + "="*50)
    logger.info("Test 5: Missing session token")
    logger.info("="*50)
    
    success5 = test_missing_session_token()
    
    # Test 6: Invalid session_id
    logger.info("\n" + "="*50)
    logger.info("Test 6: Invalid session_id")
    logger.info("="*50)
    
    success6 = test_invalid_session_id()
    
    # Summary
    logger.info("\n" + "="*50)
    logger.info("TEST SUMMARY")
    logger.info("="*50)
    
    logger.info(f"Valid request test: {'✅ PASSED' if success1 else '❌ FAILED'}")
    logger.info(f"Invalid API key test: {'✅ PASSED' if success2 else '❌ FAILED'}")
    logger.info(f"Missing API key test: {'✅ PASSED' if success3 else '❌ FAILED'}")
    logger.info(f"Missing session_id test: {'✅ PASSED' if success4 else '❌ FAILED'}")
    logger.info(f"Missing session token test: {'✅ PASSED' if success5 else '❌ FAILED'}")
    logger.info(f"Invalid session_id test: {'✅ PASSED' if success6 else '❌ FAILED'}")
    
    results = [success1, success2, success3, success4, success5, success6]
    total_passed = sum(results)
    logger.info(f"\nOverall: {total_passed}/{len(results)} tests passed")
    
    if all(results):
        logger.info("🎉 All tests passed!")
        return 0
    logger.error("❌ Some tests failed. Check the logs above for details.")
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
