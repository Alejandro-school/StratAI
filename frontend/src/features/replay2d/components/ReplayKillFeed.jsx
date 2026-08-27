import { Crosshair, EyeOff, Layers3 } from "lucide-react";
import { weaponIconPath } from "../domain/weaponPresentation";

export function ReplayKillFeed({ events, tick, tickRate }) {
  const kills = events
    .filter((event) => event.type === "kill" && event.tick <= tick && tick - event.tick < tickRate * 5)
    .slice(-5)
    .reverse();
  if (!kills.length) return null;
  return (
    <ol className="r2-kill-feed" aria-label="Eliminaciones recientes">
      {kills.map((kill) => (
        <li key={kill.id} className={`r2-kill-row r2-killer-${kill.killer_team?.toLowerCase() || "unknown"}`}>
          <span className={`r2-kill-player ${kill.killer_team?.toLowerCase()}`}>{kill.killer_name || "Jugador"}</span>
          {kill.assister_name && (
            <span className={`r2-kill-assist ${kill.assister_team?.toLowerCase() || kill.killer_team?.toLowerCase()}`}>
              + {kill.assister_name}
            </span>
          )}
          <span className="r2-kill-weapon" aria-label={kill.weapon || "arma"}>
            {kill.wallbang && <Layers3 className="r2-kill-modifier" aria-label="Penetración" />}
            {weaponIconPath(kill.weapon) ? (
              <img src={weaponIconPath(kill.weapon)} alt="" />
            ) : (
              <span className="r2-kill-weapon-fallback">{kill.weapon || "arma"}</span>
            )}
            {kill.noscope && <EyeOff className="r2-kill-modifier" aria-label="Sin mira" />}
            {kill.headshot && (
              <span className="r2-kill-hs" aria-label="Disparo a la cabeza">
                <Crosshair className="r2-kill-headshot" />
                HS
              </span>
            )}
          </span>
          <span className={`r2-kill-player ${kill.victim_team?.toLowerCase()}`}>{kill.victim_name || "Jugador"}</span>
        </li>
      ))}
    </ol>
  );
}
