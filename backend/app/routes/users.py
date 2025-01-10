from fastapi import APIRouter, Request, HTTPException, status, Response
from app.utils.steam_openid import get_openid_redirect_url, validate_steam_login

router = APIRouter()

@router.get("/auth/steam/login")
def steam_login(request: Request):
    """
    Crea la URL de Steam OpenID y redirige al usuario a Steam.
    """
    # Generar la URL callback a la que Steam mandará la respuesta.
    # En local, puede ser: http://localhost:8000/auth/steam/callback
    # En producción, tu dominio real.
    callback_url = str(request.url_for("steam_callback"))
    
    # Construimos la URL de Steam
    redirect_url = get_openid_redirect_url(callback_url)
    
    # Redirigimos al usuario
    return Response(status_code=status.HTTP_302_FOUND, headers={"Location": redirect_url})


@router.get("/auth/steam/callback", name="steam_callback")
def steam_callback(request: Request):
    """
    Steam redirige aquí tras la autenticación. 
    Validamos la firma y sacamos la SteamID.
    """
    steam_id = validate_steam_login(request)
    if not steam_id:
        raise HTTPException(status_code=400, detail="No se pudo validar la cuenta de Steam")

    # Aquí ya tienes la SteamID. Por ejemplo: 76561198000030757
    # Normalmente crearías una sesión, un JWT, etc.
    # Por simplicidad, solo devolvemos un JSON con la steam_id.
    return {"msg": "Login exitoso", "steam_id": steam_id}
