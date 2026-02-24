from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from mangum import Mangum

app = FastAPI(title="debugjois.dev API")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/")
async def root():
    return {"message": "Hello from debugjois.dev Lambda!"}


@app.get("/health")
async def health(request: Request):
    event = request.scope.get("aws.event", {})
    claims = event.get("requestContext", {}).get("authorizer", {}).get("jwt", {}).get("claims", {})
    return {"status": "ok", "email": claims.get("email")}


handler = Mangum(app)
