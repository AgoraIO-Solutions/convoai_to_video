import requests
import json
import logging

# Configuration for local testing
API_ENDPOINT = "http://localhost:8764/session/start"  # Points to mock server
API_KEY = "YOUR_API_KEY"

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def test_session_start_endpoint():
    """Test the session start POST endpoint"""
    
    # Prepare test payload (API key moved to headers)
    payload = {
        "avatar_id": "16cb73e7de08",
        "quality": "high",
        "version": "v1",
        "video_encoding": "H264",
        "agora_settings": {
            "app_id": "dllkSlkdmmppollalepls",
            "token": "lkmmopplek",
            "channel": "room1",
            "uid": "333",
            "enable_string_uid": False
        }
    }
    
    # API key now in headers for better security
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": API_KEY
    }
    
    logger.info(f"Testing endpoint: {API_ENDPOINT}")
    logger.info(f"Headers (API key masked): {dict(headers, **{'x-api-key': '***masked***'})}")
    logger.info(f"Payload: {json.dumps(payload, indent=2)}")
    
    try:
        # Send POST request
        response = requests.post(
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
    
    required_fields = ["websocket_address", "session_token"]
    missing_fields = []
    
    for field in required_fields:
        if field not in data:
            missing_fields.append(field)
        else:
            logger.info(f"✅ Found required field: {field}")
    
    if missing_fields:
        logger.error(f"❌ Missing required fields: {missing_fields}")
        return False
    
    # Verify websocket_address format
    websocket_address = data["websocket_address"]
    expected_websocket = "ws://localhost:8765"
    
    if websocket_address != expected_websocket:
        logger.error(f"❌ Unexpected websocket_address: {websocket_address}")
        logger.error(f"Expected: {expected_websocket}")
        return False
    else:
        logger.info(f"✅ Correct websocket_address: {websocket_address}")
    
    # Verify session_token is not empty
    session_token = data["session_token"]
    if not session_token or len(session_token.strip()) == 0:
        logger.error("❌ session_token is empty")
        return False
    else:
        logger.info(f"✅ session_token present (length: {len(session_token)} chars)")
        # Log first and last few characters for debugging
        if len(session_token) > 20:
            logger.info(f"Token preview: {session_token[:10]}...{session_token[-10:]}")
    
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
        "avatar_id": "test_avatar",
        "quality": "high",
        "version": "v1",
        "video_encoding": "H264",
        "agora_settings": {
            "app_id": "test_app_id",
            "token": "test_token",
            "channel": "test_channel",
            "uid": "123",
            "enable_string_uid": False
        }
    }
    
    # Invalid API key in headers
    headers = {
        "accept": "application/json",
        "content-type": "application/json",
        "x-api-key": "INVALID_API_KEY"
    }
    
    try:
        response = requests.post(API_ENDPOINT, headers=headers, json=payload, timeout=30)
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
        "avatar_id": "test_avatar",
        "quality": "high",
        "version": "v1",
        "video_encoding": "H264",
        "agora_settings": {
            "app_id": "test_app_id",
            "token": "test_token",
            "channel": "test_channel",
            "uid": "123",
            "enable_string_uid": False
        }
    }
    
    # Headers without API key
    headers = {
        "accept": "application/json",
        "content-type": "application/json"
        # Missing x-api-key header
    }
    
    try:
        response = requests.post(API_ENDPOINT, headers=headers, json=payload, timeout=30)
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


def test_malformed_payload():
    """Test the endpoint with malformed payload"""
    logger.info("\n" + "="*50)
    logger.info("Testing with malformed payload (missing required field)...")
    
    # Missing avatar_id field
    payload = {
        "quality": "high",
        "version": "v1",
        "video_encoding": "H264",
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
        response = requests.post(API_ENDPOINT, headers=headers, json=payload, timeout=30)
        logger.info(f"Response status code: {response.status_code}")
        
        if response.status_code == 400:
            logger.info("✅ Correctly received 400 Bad Request for malformed payload")
            return True
        else:
            logger.warning(f"⚠️ Expected 400 but got {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Error during malformed payload test: {e}")
        return False


def main():
    """Run all tests"""
    logger.info("=" * 60)
    logger.info("SESSION START ENDPOINT TEST (LOCAL)")
    logger.info("=" * 60)
    logger.info("NOTE: Make sure session_test_receiver.py is running on port 8764")
    logger.info("=" * 60)
    
    # Test 1: Valid request
    logger.info("\n" + "="*50)
    logger.info("Test 1: Valid API request")
    logger.info("="*50)
    
    success1 = test_session_start_endpoint()
    
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
    
    # Test 4: Malformed payload
    logger.info("\n" + "="*50)
    logger.info("Test 4: Malformed payload")
    logger.info("="*50)
    
    success4 = test_malformed_payload()
    
    # Summary
    logger.info("\n" + "="*50)
    logger.info("TEST SUMMARY")
    logger.info("="*50)
    
    logger.info(f"Valid request test: {'✅ PASSED' if success1 else '❌ FAILED'}")
    logger.info(f"Invalid API key test: {'✅ PASSED' if success2 else '❌ FAILED'}")
    logger.info(f"Missing API key test: {'✅ PASSED' if success3 else '❌ FAILED'}")
    logger.info(f"Malformed payload test: {'✅ PASSED' if success4 else '❌ FAILED'}")
    
    total_passed = sum([success1, success2, success3, success4])
    logger.info(f"\nOverall: {total_passed}/4 tests passed")
    
    if total_passed == 4:
        logger.info("🎉 All tests passed!")
    else:
        logger.info("⚠️ Some tests failed. Check the logs above for details.")


if __name__ == "__main__":
    main()