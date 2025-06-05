from fastapi import APIRouter, UploadFile, File, HTTPException
import json
from app.services.buycoach import get_buy_advice

router = APIRouter(prefix="/buy-coach", tags=["Buy Coach"])

@router.post("/")
async def advise(file: UploadFile = File(...)):
    if file.filename != "economy.json":
        raise HTTPException(status_code=400, detail="Debe subir economy.json")
    eco_json = json.loads(await file.read())
    return get_buy_advice(eco_json)
