defmodule ElixirService.Router do
  use Plug.Router
  require Logger

  plug :match
  plug :dispatch

  defp structured_log(level, message, fields \\ %{}) do
    entry = %{
      timestamp: DateTime.utc_now() |> DateTime.to_iso8601(),
      level: level,
      service: "elixir-service",
      message: message,
      fields: fields
    }
    |> Jason.encode!()
    
    Logger.info(entry)
  end

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

  get "/health" do
    fields = %{
      remote_addr: get_remote_addr(conn),
      method: conn.method,
      path: conn.request_path
    }
    
    structured_log("INFO", "Health check requested", fields)
    
    body = ~s({"status":"healthy","service":"elixir"})
    
    structured_log("INFO", "Health check completed", Map.put(fields, :status, "healthy"))
    
    conn
    |> put_resp_content_type("application/json")
    |> send_resp(200, body)
  end

  get "/" do
    fields = %{
      remote_addr: get_remote_addr(conn),
      method: conn.method,
      path: conn.request_path
    }
    structured_log("INFO", "Root endpoint accessed", fields)
    
    send_resp(conn, 200, "Elixir Service is running!")
  end

  match _ do
    send_resp(conn, 404, "Not found")
  end
end
