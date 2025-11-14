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

// Structured logging function
interface LogEntry {
  timestamp: string;
  level: string;
  service: string;
  message: string;
  fields?: Record<string, any>;
}

function structuredLog(level: string, message: string, fields?: Record<string, any>): void {
  const entry: LogEntry = {
    timestamp: new Date().toISOString(),
    level,
    service: SERVICE_NAME,
    message,
    ...(fields && { fields }),
  };
  console.log(JSON.stringify(entry));
}

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

// Middleware for request logging and metrics
app.use((req: Request, res: Response, next: Function) => {
  const start = Date.now();
  
  res.on('finish', () => {
    const duration = (Date.now() - start) / 1000;
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
  
  const fields = {
    remote_addr: req.ip || req.socket.remoteAddress,
    method: req.method,
    path: req.path,
  };
  
  structuredLog('INFO', 'Health check requested', fields);
  
  res.status(200).json({
    status: 'healthy',
    service: 'typescript',
  });
  
  const fieldsCompleted = {
    ...fields,
    status: 'healthy',
  };
  structuredLog('INFO', 'Health check completed', fieldsCompleted);
  
  span.end();
});

app.get('/', (req: Request, res: Response) => {
  const fields = {
    remote_addr: req.ip || req.socket.remoteAddress,
    method: req.method,
    path: req.path,
  };
  structuredLog('INFO', 'Root endpoint accessed', fields);
  
  res.status(200).send('TypeScript Service is running!');
});

app.get('/metrics', async (req: Request, res: Response) => {
  res.set('Content-Type', register.contentType);
  res.end(await register.metrics());
});

app.listen(PORT, () => {
  structuredLog('INFO', 'TypeScript service starting', { port: PORT });
});
