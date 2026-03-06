from unittest.mock import MagicMock

from fastapi.testclient import TestClient

from app.main import app, get_email_from_request

client = TestClient(app)


def test_root():
    response = client.get("/")
    assert response.status_code == 200
    assert response.json() == {"message": "Hello from debugjois.dev Lambda!"}


def test_health_no_jwt_context():
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json() == {"status": "ok", "email": None}


def test_get_email_without_event():
    mock_req = MagicMock()
    mock_req.scope = {}
    assert get_email_from_request(mock_req) is None


def test_get_email_with_jwt_claims():
    mock_req = MagicMock()
    mock_req.scope = {
        "aws.event": {
            "requestContext": {
                "authorizer": {"jwt": {"claims": {"email": "test@example.com"}}}
            }
        }
    }
    assert get_email_from_request(mock_req) == "test@example.com"
