defmodule ElixirService.Application do
  @moduledoc false
  use Application

  @impl true
  def start(_type, _args) do
    # Initialize OpenTelemetry
    init_opentelemetry()

    children = [
      {Plug.Cowboy, scheme: :http, plug: ElixirService.Router, options: [port: 4000]}
    ]

    opts = [strategy: :one_for_one, name: ElixirService.Supervisor]
    Supervisor.start_link(children, opts)
  end

  defp init_opentelemetry do
    # OpenTelemetry will use environment variables automatically
    # OTEL_EXPORTER_OTLP_ENDPOINT and OTEL_SERVICE_NAME are set in docker-compose.yml
    :ok
  end
end
