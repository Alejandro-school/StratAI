from fastapi import APIRouter, HTTPException, UploadFile, status
from pydantic import BaseModel
import httpx

router = APIRouter()

class StatsResponse(BaseModel):
    kills: int
    assists: int
    deaths: int
    headshots: int
    rounds_played: int

@router.post("/stats/{match_id}", response_model=StatsResponse)
async def get_stats(match_id: str, file: UploadFile):
    try:
        async with httpx.AsyncClient() as client:
            files = {
                "file": (
                    file.filename,
                    await file.read(),
                    file.content_type
                )
            }
            response = await client.post("http://localhost:8080/process-demo", files=files)

        if response.status_code != 200:
            raise HTTPException(
                status_code=response.status_code,
                detail=f"Error en microservicio Go: {response.text}"
            )

        return response.json()  # Devuelve { kills, assists, ... }

    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=str(e)
        )
