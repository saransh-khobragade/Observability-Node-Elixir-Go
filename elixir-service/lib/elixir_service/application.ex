defmodule ElixirService.Application do
  @moduledoc false
  use Application
  require Logger

  @impl true
  def start(_type, _args) do
    # Initialize OpenTelemetry
    init_opentelemetry()

    children = [
      {Plug.Cowboy, scheme: :http, plug: ElixirService.Router, options: [port: 4000]}
    ]

    opts = [strategy: :one_for_one, name: ElixirService.Supervisor]
    
    case Supervisor.start_link(children, opts) do
      {:ok, _pid} = result ->
        log_startup()
        result
      error ->
        error
    end
  end

  defp init_opentelemetry do
    otlp_endpoint = System.get_env("OTEL_EXPORTER_OTLP_ENDPOINT") || "http://otel-collector:4317"
    
    # OpenTelemetry will use environment variables automatically
    # OTEL_EXPORTER_OTLP_ENDPOINT and OTEL_SERVICE_NAME are set in docker-compose.yml
    log_opentelemetry_init(otlp_endpoint)
    :ok
  end

  defp log_startup do
    entry = %{
      timestamp: DateTime.utc_now() |> DateTime.to_iso8601(),
      level: "INFO",
      service: "elixir-service",
      message: "Elixir service started",
      fields: %{
        port: 4000
      }
    }
    |> Jason.encode!()
    
    # Write directly to :stdio device (like TypeScript console.log) to avoid Logger formatting
    IO.puts(:stdio, entry)
  end

  defp log_opentelemetry_init(otlp_endpoint) do
    entry = %{
      timestamp: DateTime.utc_now() |> DateTime.to_iso8601(),
      level: "INFO",
      service: "elixir-service",
      message: "OpenTelemetry SDK initialized",
      fields: %{
        otlp_endpoint: otlp_endpoint
      }
    }
    |> Jason.encode!()
    
    # Write directly to :stdio device (like TypeScript console.log) to avoid Logger formatting
    IO.puts(:stdio, entry)
  end
end
