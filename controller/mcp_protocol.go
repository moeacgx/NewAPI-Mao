package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type mcpRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

var mcpRPCIntegerID = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

func mcpRPCError(c *gin.Context, status int, id json.RawMessage, code int, message string) {
	if len(id) == 0 {
		id = json.RawMessage(`null`)
	}
	c.JSON(status, gin.H{"jsonrpc": "2.0", "id": id, "error": gin.H{"code": code, "message": message}})
}

// MCPHandler 实现无状态 Streamable HTTP JSON 模式；通知不执行有副作用工具。
func MCPHandler(dispatch http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			mcpRPCError(c, http.StatusRequestEntityTooLarge, nil, -32600, "Request body too large")
			return
		}
		var value any
		if err := common.Unmarshal(raw, &value); err != nil {
			mcpRPCError(c, http.StatusBadRequest, nil, -32700, "Parse error")
			return
		}
		object, ok := value.(map[string]any)
		if !ok || object["jsonrpc"] != "2.0" {
			mcpRPCError(c, http.StatusBadRequest, nil, -32600, "Invalid Request")
			return
		}
		var request mcpRPCRequest
		if err := common.Unmarshal(raw, &request); err != nil {
			mcpRPCError(c, http.StatusBadRequest, nil, -32600, "Invalid Request")
			return
		}
		if len(request.ID) != 0 {
			kind := common.GetJsonType(request.ID)
			if kind != "number" && kind != "string" || len(request.ID) > 256 || kind == "number" && !mcpRPCIntegerID.Match(bytes.TrimSpace(request.ID)) {
				mcpRPCError(c, http.StatusBadRequest, nil, -32600, "Invalid request ID")
				return
			}
		}
		if len(request.Params) != 0 && !bytes.HasPrefix(bytes.TrimSpace(request.Params), []byte("{")) {
			mcpRPCError(c, http.StatusBadRequest, request.ID, -32602, "Params must be an object")
			return
		}
		if request.Method == "" {
			// 无状态服务不发起客户端请求；有效响应只确认接收。
			_, result := object["result"]
			_, rpcError := object["error"]
			if len(request.ID) != 0 && result != rpcError {
				if rpcError {
					payload, ok := object["error"].(map[string]any)
					code, number := payload["code"].(float64)
					_, message := payload["message"].(string)
					if !ok || !number || !message || math.Trunc(code) != code {
						mcpRPCError(c, http.StatusBadRequest, request.ID, -32600, "Invalid response error")
						return
					}
				}
				c.Status(http.StatusAccepted)
				return
			}
			mcpRPCError(c, http.StatusBadRequest, request.ID, -32600, "Invalid Request")
			return
		}
		if _, exists := object["result"]; exists {
			mcpRPCError(c, http.StatusBadRequest, request.ID, -32600, "Invalid Request")
			return
		}
		if _, exists := object["error"]; exists {
			mcpRPCError(c, http.StatusBadRequest, request.ID, -32600, "Invalid Request")
			return
		}
		if len(request.ID) == 0 {
			c.Status(http.StatusAccepted)
			return
		}
		identity := middleware.MCPIdentity(c)
		var result any
		switch request.Method {
		case "initialize":
			var params struct {
				ProtocolVersion string         `json:"protocolVersion"`
				Capabilities    map[string]any `json:"capabilities"`
				ClientInfo      struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				} `json:"clientInfo"`
			}
			if common.Unmarshal(request.Params, &params) != nil || params.ProtocolVersion == "" || params.Capabilities == nil || params.ClientInfo.Name == "" || params.ClientInfo.Version == "" {
				mcpRPCError(c, http.StatusOK, request.ID, -32602, "Invalid initialize params")
				return
			}
			version := params.ProtocolVersion
			if !service.MCPProtocolSupported(version) {
				version = "2025-11-25"
			}
			result = gin.H{"protocolVersion": version, "capabilities": gin.H{"tools": gin.H{}}, "serverInfo": gin.H{"name": "NewAPI-Mao", "version": "1.0.0"}, "instructions": service.MCPMoneyInstructions}
		case "ping":
			result = gin.H{}
		case "tools/list":
			var params struct {
				Cursor string `json:"cursor"`
			}
			if len(request.Params) != 0 && common.Unmarshal(request.Params, &params) != nil || params.Cursor != "" {
				mcpRPCError(c, http.StatusOK, request.ID, -32602, "Invalid cursor")
				return
			}
			result = gin.H{"tools": service.MCPVisibleTools(identity.User, identity.Key)}
		case "tools/call":
			var params struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			if common.Unmarshal(request.Params, &params) != nil || strings.TrimSpace(params.Name) == "" || len(params.Name) > 100 {
				mcpRPCError(c, http.StatusOK, request.ID, -32602, "Invalid tool call")
				return
			}
			result = service.ExecuteMCPTool(c.Request.Context(), identity, params.Name, params.Arguments, dispatch)
		default:
			mcpRPCError(c, http.StatusOK, request.ID, -32601, "Method not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}
}
