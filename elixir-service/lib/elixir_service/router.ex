defmodule ElixirService.Router do
  use Plug.Router

  # Logging middleware must be before :match and :dispatch
  plug :log_request
  plug :match
  plug :dispatch

  # Structured logging function - Consistent format across all services
  # Matches TypeScript/Go structure: core fields at top level, request fields nested in "fields" object
  # Outputs raw JSON to stdout (like TypeScript console.log) for consistent parsing
  defp log(level, message, additional_fields \\ %{}) do
    entry = %{
      timestamp: DateTime.utc_now() |> DateTime.to_iso8601(),
      level: String.upcase(level),
      service: "elixir-service",
      message: message,
      fields: additional_fields
    }
    |> Jason.encode!()
    
    # Write directly to :stdio device (like TypeScript console.log) to avoid Logger formatting
    # This ensures raw JSON output without any Logger wrapper
    IO.puts(:stdio, entry)
    :ok
  end

  # Convenience methods
  defp log_info(message, fields \\ %{}), do: log("INFO", message, fields)
  defp log_warn(message, fields \\ %{}), do: log("WARN", message, fields)
  defp log_error(message, fields \\ %{}), do: log("ERROR", message, fields)
  defp log_debug(message, fields \\ %{}), do: log("DEBUG", message, fields)

  defp get_remote_addr(conn) do
    case Plug.Conn.get_req_header(conn, "x-forwarded-for") do
      [addr | _] -> addr
      [] -> 
        case Plug.Conn.get_req_header(conn, "x-real-ip") do
          [addr | _] -> addr
          [] -> 
            case conn.remote_ip do
              {a, b, c, d} -> "#{a}.#{b}.#{c}.#{d}"
              _ -> "unknown"
            end
        end
    end
  end

  defp log_request(conn, _opts) do
    start_time = System.monotonic_time(:millisecond)
    
    # Log incoming request
    log_info("Incoming HTTP request", %{
      remote_addr: get_remote_addr(conn),
      method: conn.method,
      path: conn.request_path,
      user_agent: List.first(Plug.Conn.get_req_header(conn, "user-agent")) || "unknown"
    })
    
    # Register callback to log response
    Plug.Conn.register_before_send(conn, fn conn ->
      duration = (System.monotonic_time(:millisecond) - start_time) / 1000.0
      status = conn.status
      
      log_level = cond do
        status >= 500 -> "ERROR"
        status >= 400 -> "WARN"
        true -> "INFO"
      end
      
      log_func = case log_level do
        "ERROR" -> &log_error/2
        "WARN" -> &log_warn/2
        _ -> &log_info/2
      end
      
      log_func.("HTTP request completed", %{
        remote_addr: get_remote_addr(conn),
        method: conn.method,
        path: conn.request_path,
        status: status,
        duration_seconds: duration
      })
      
      conn
    end)
  end

  get "/health" do
    body = ~s({"status":"healthy","service":"elixir"})
    
    conn
    |> put_resp_content_type("application/json")
    |> send_resp(200, body)
  end

  get "/" do
    send_resp(conn, 200, "Elixir Service is running!")
  end

  match _ do
    send_resp(conn, 404, "Not found")
  end
end
