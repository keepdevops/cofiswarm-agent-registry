#pragma once

#include <string>
#include <vector>

struct Agent {
    std::string name;
    int port;
    int read_timeout_secs;
    int max_tokens;
    std::string system_prompt;
    std::string description;
    std::vector<std::string> tags; // e.g. {"planning"}, {"coding"}, {"review"}
    std::string backend;
    std::string engine; // "llama" (default), "mlx", or "docker"
    std::string model;  // model ID — sent in request body for docker/vllm
    /// MS-161/MS-68: "http" (default), "inproc"/"inprocess", or "auto".
    std::string dispatch = "http";
    /// Quantization label for registry key, e.g. "4bit". Empty → "default".
    std::string quant;
    /// Request Metal flash-attention when embed path supports it (Phase 2).
    bool use_flash_attention = false;

    // Speculative decoding config (llama-server only). The coordinator does
    // not act on these directly — they are recorded so the launch script can
    // pass --model-draft / --draft-max when starting the agent's server, and
    // so /api/agents reports them. Empty draft_model = no speculative decode.
    std::string draft_model;
    int draft_max = 0;

    /// Deployed model context length from swarm-config `context` (inference window).
    int context_window = 8192;

    /// Max simultaneous in-flight requests to this agent's port.
    /// 0 = unlimited (default for llama/vllm).
    /// 1 = serialized (default for mlx, which cannot handle concurrent requests).
    /// >1 = counted semaphore (useful for vllm with known concurrency limits).
    int max_concurrency = 0;

    /// Hard input truncation in tokens (approximate: 4 chars ≈ 1 token).
    /// 0 = no cap (default). Set via agents[].max_input_tokens in swarm-config.
    int max_input_tokens  = 0;
    /// Generation length cap. Maps to num_predict (llama) or max_tokens override.
    /// 0 = use agent.max_tokens (default).
    int max_output_tokens = 0;

    /// Per-port KV token budget: max total tokens (prompt+completion) that may
    /// be in-flight simultaneously across all agents sharing this port.
    /// 0 = disabled (default). Set via coordinator.json kv_token_budget or
    /// agents[].kv_token_budget in swarm-config.
    int kv_token_budget = 0;

    /// Inference backend override: "" (legacy), "auto", "llama_metal", "python_mlx".
    /// Opt-in routing (MATRIX_BACKEND_ROUTING) applies in sequential modes only.
    std::string inference_backend;
};
