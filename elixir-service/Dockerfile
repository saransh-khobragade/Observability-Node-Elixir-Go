FROM elixir:1.15-alpine

# Install curl for healthcheck
RUN apk add --no-cache curl

WORKDIR /app

# Install hex and rebar
RUN mix local.hex --force && \
    mix local.rebar --force

# Copy mix files (mix.lock is optional)
COPY mix.exs ./
COPY mix.lock* ./

# Install dependencies
RUN mix deps.get

# Copy application code
COPY . .

# Compile the application
RUN mix compile

# Expose port
EXPOSE 4000

# Run the application
CMD ["mix", "run", "--no-halt"]

