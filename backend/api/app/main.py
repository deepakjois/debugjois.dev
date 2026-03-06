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


def get_email_from_request(request: Request) -> str | None:
    """Extract user email from API Gateway JWT authorizer claims, or None locally."""
    event = request.scope.get("aws.event", {})
    claims = (
        event.get("requestContext", {})
        .get("authorizer", {})
        .get("jwt", {})
        .get("claims", {})
    )
    return claims.get("email")


@app.get("/")
async def root():
    return {"message": "Hello from debugjois.dev Lambda!"}


@app.get("/health")
async def health(request: Request):
    return {"status": "ok", "email": get_email_from_request(request)}


handler = Mangum(app)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run("app.main:app", host="0.0.0.0", port=8000, reload=True)
