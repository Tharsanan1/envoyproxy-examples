# Service Setup and Testing

## Starting the Services

To start the services, run the following commands:

```bash
chmod +x ./start.sh
./start.sh
```

## Viewing Envoy Proxy Logs

To listen for the Envoy proxy logs, run the following command:

```bash
docker-compose logs -f proxy
```

## Sending Requests to Envoy

Open a separate terminal and run the following command to send a request to Envoy:

```bash
curl -v localhost:10000 2>&1 | grep rsp-header-from-go
```

### Expected Output

You will see the Google HTML content along with a "pong" response from Redis in the envoy proxy logs.
