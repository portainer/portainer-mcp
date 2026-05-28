"""Tests for the remote_auth module."""

from unittest.mock import patch, MagicMock

import httpx
import pytest

from portainer_mcp import remote_auth


class TestGetClientToken:
    def test_returns_none_when_no_request(self):
        """Should return None when no HTTP request is in flight."""
        with patch(
            "portainer_mcp.remote_auth.get_http_request",
            side_effect=RuntimeError("no request"),
        ):
            assert remote_auth.get_client_token() is None

    def test_extracts_custom_header(self):
        """X-Portainer-API-Key takes priority."""
        request = MagicMock()
        request.headers = {"x-portainer-api-key": "ptr_custom", "authorization": "Bearer ptr_bearer"}
        with patch("portainer_mcp.remote_auth.get_http_request", return_value=request):
            assert remote_auth.get_client_token() == "ptr_custom"

    def test_extracts_bearer_token(self):
        """Falls back to Authorization: Bearer."""
        request = MagicMock()
        request.headers = {"authorization": "Bearer ptr_from_bearer"}
        with patch("portainer_mcp.remote_auth.get_http_request", return_value=request):
            assert remote_auth.get_client_token() == "ptr_from_bearer"

    def test_returns_none_without_auth_headers(self):
        """Returns None when no relevant headers are present."""
        request = MagicMock()
        request.headers = {"user-agent": "test"}
        with patch("portainer_mcp.remote_auth.get_http_request", return_value=request):
            assert remote_auth.get_client_token() is None


class TestRemoteAuthVerifier:
    @pytest.fixture
    def verifier(self):
        return remote_auth.RemoteAuthVerifier()

    async def test_accepts_non_empty_token(self, verifier):
        with patch("portainer_mcp.remote_auth.request_context.snapshot", return_value={}):
            result = await verifier.verify_token("ptr_valid_token_here_1234567890")
            assert result is not None
            assert result.client_id == "remote-user"

    async def test_rejects_empty_token(self, verifier):
        with patch("portainer_mcp.remote_auth.request_context.snapshot", return_value={}):
            result = await verifier.verify_token("")
            assert result is None

    async def test_rejects_whitespace_token(self, verifier):
        with patch("portainer_mcp.remote_auth.request_context.snapshot", return_value={}):
            result = await verifier.verify_token("   ")
            assert result is None


class TestInjectTokenHook:
    def test_injects_token_from_context(self):
        """Hook should set X-API-KEY from the client's token."""
        request = httpx.Request("GET", "http://portainer.local/api/endpoints")
        with patch("portainer_mcp.remote_auth.get_client_token", return_value="ptr_injected"):
            remote_auth.inject_token_hook(request)
        assert request.headers["x-api-key"] == "ptr_injected"

    def test_no_injection_when_no_token(self):
        """Hook should not set header when no client token is available."""
        request = httpx.Request("GET", "http://portainer.local/api/endpoints")
        with patch("portainer_mcp.remote_auth.get_client_token", return_value=None):
            remote_auth.inject_token_hook(request)
        assert "x-api-key" not in request.headers
