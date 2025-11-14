import express, { Request, Response } from 'express';
import { Registry, Counter, Histogram } from 'prom-client';
import * as opentelemetry from '@opentelemetry/api';
import { NodeSDK } from '@opentelemetry/sdk-node';
import { Resource } from '@opentelemetry/resources';
import { SemanticResourceAttributes } from '@opentelemetry/semantic-conventions';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';

const app = express();
const PORT = 3000;
const SERVICE_NAME = 'typescript-service';

// Prometheus metrics
const register = new Registry();
const httpRequestsTotal = new Counter({
  name: 'http_requests_total',
  help: 'Total number of HTTP requests',
  labelNames: ['method', 'endpoint', 'status'],
  registers: [register],
});

const httpRequestDuration = new Histogram({
  name: 'http_request_duration_seconds',
  help: 'HTTP request duration in seconds',
  labelNames: ['method', 'endpoint'],
  registers: [register],
});

// Structured logging function - Consistent format across all services
// Core fields at top level, request/context fields nested in "fields" object
interface LogEntry {
  timestamp: string;
  level: string;
  service: string;
  message: string;
  fields?: Record<string, any>;
}

function log(level: string, message: string, additionalFields?: Record<string, any>): void {
  const entry: LogEntry = {
    timestamp: new Date().toISOString(),
    level: level.toUpperCase(),
    service: SERVICE_NAME,
    message,
    ...(additionalFields && Object.keys(additionalFields).length > 0 ? { fields: additionalFields } : {}),
  };
  console.log(JSON.stringify(entry));
}

// Convenience methods
const logger = {
  info: (message: string, fields?: Record<string, any>) => log('INFO', message, fields),
  warn: (message: string, fields?: Record<string, any>) => log('WARN', message, fields),
  error: (message: string, fields?: Record<string, any>) => log('ERROR', message, fields),
  debug: (message: string, fields?: Record<string, any>) => log('DEBUG', message, fields),
};

// Initialize OpenTelemetry
const otlpEndpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT || 'http://otel-collector:4317';

const sdk = new NodeSDK({
  resource: new Resource({
    [SemanticResourceAttributes.SERVICE_NAME]: SERVICE_NAME,
  }),
  traceExporter: undefined, // Will use OTLP exporter via environment variables
  instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();
logger.info('OpenTelemetry SDK initialized', {
  otlp_endpoint: otlpEndpoint,
});

// Middleware for request logging and metrics
app.use((req: Request, res: Response, next: Function) => {
  const start = Date.now();
  
  // Log incoming request
  logger.info('Incoming HTTP request', {
    remote_addr: req.ip || req.socket.remoteAddress,
    method: req.method,
    path: req.path,
    user_agent: req.get('user-agent'),
  });
  
  res.on('finish', () => {
    const duration = (Date.now() - start) / 1000;
    const level = res.statusCode >= 500 ? 'ERROR' : res.statusCode >= 400 ? 'WARN' : 'INFO';
    
    // Log response
    logger[level.toLowerCase() as 'info' | 'warn' | 'error']('HTTP request completed', {
      remote_addr: req.ip || req.socket.remoteAddress,
      method: req.method,
      path: req.path,
      status: res.statusCode,
      duration_seconds: duration,
    });
    
    // Update metrics
    httpRequestsTotal.inc({
      method: req.method,
      endpoint: req.path,
      status: res.statusCode.toString(),
    });
    httpRequestDuration.observe(
      {
        method: req.method,
        endpoint: req.path,
      },
      duration
    );
  });
  
  next();
});

app.use(express.json());

app.get('/health', (req: Request, res: Response) => {
  const tracer = opentelemetry.trace.getTracer(SERVICE_NAME);
  const span = tracer.startSpan('health_check');
  
  try {
    res.status(200).json({
      status: 'healthy',
      service: 'typescript',
    });
    span.end();
  } catch (error) {
    logger.error('Health check failed', {
      error: error instanceof Error ? error.message : String(error),
      stack: error instanceof Error ? error.stack : undefined,
    });
    span.end();
    res.status(500).json({ status: 'unhealthy', service: 'typescript' });
  }
});

app.get('/', (req: Request, res: Response) => {
  res.status(200).send('TypeScript Service is running!');
});

app.get('/metrics', async (req: Request, res: Response) => {
  res.set('Content-Type', register.contentType);
  res.end(await register.metrics());
});

app.listen(PORT, () => {
  logger.info('TypeScript service started', { port: PORT });
});

// Handle uncaught errors
process.on('uncaughtException', (error) => {
  logger.error('Uncaught exception', {
    error: error.message,
    stack: error.stack,
  });
  process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
  logger.error('Unhandled promise rejection', {
    reason: reason instanceof Error ? reason.message : String(reason),
    stack: reason instanceof Error ? reason.stack : undefined,
  });
});
