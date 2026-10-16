"""Reference-only replay readings: facts and prose share the same dated value.

The model selects evidence, never supplies factual text. Conditional principles
are retained as review aids; this does not evaluate tactical interpretation.
Legacy CS3 receipts deliberately keep their original consultation protocol.
"""
from copy import deepcopy

from ai_coach.interpretation import context_comparison as comparison
from ai_coach.support.anchors import require
from ai_coach.interpretation.replay_factual_validation import NATIVE_EQUIPMENT

VERSION = "stratai.replay_factual_reading@1"
PROMPT = """Select evidence for a Spanish contextual reading. Do not write prose.
Use the supplied reference_selection schema. situation selects current fact IDs;
evolution selects historical/event IDs; uncertainty selects locally uncertain
facts. interpretation selects only an existing conditional principle and its
consulted evidence IDs. Selecting a principle never establishes its premise.
Earlier delivered start facts can be cited directly; other samples need a query.
Facts, times, certainty and text are rendered and checked by the application.
"""


def alias(value):
    return f"P{int(value):02d}" if value is not None else "participante no identificado"


def number(value):
    # Keep tick fractions/native values; general-format's six digits silently
    # changed e.g. 114.9375 into an apparently exact countdown of 114.938.
    return str(int(value)) if type(value) is float and value.is_integer() else str(value)


def describe(fact):
    """Closed descriptions of the facts already supplied, without geometry guesses."""
    key, value, who = fact["key"], fact["value"], fact["subject"] or "Equipo"
    if key == "objective":
        if not value:
            return "Estado del C4 no disponible."
        state = value.get("state")
        states = {"carried": f"C4 transportado por {alias(value.get('carrier_id'))}.",
                  "dropped": "C4 en el suelo.", "planting": "Intento de plantado del C4 en curso.",
                  "planted": "C4 plantado.", "defusing": "Intento de desactivación del C4 en curso.",
                  "defused": "C4 desactivado.", "exploded": "C4 explotado.",
                  "resolved": "Objetivo de la ronda resuelto; este estado no identifica por sí solo la causa.",
                  "unknown": "Estado del C4 desconocido."}
        return states.get(state, "Estado del C4 no acreditado.") + (
            f" Posición observada: ({number(value['x'])}, {number(value['y'])})."
            if value.get("position_status") == "observed" and "x" in value and "y" in value
            else " Posición actual no disponible en este hecho.")
    if key == "own_player":
        text = (f"{who} está {'vivo' if value['alive'] else 'muerto'}, con {number(value.get('health'))} de salud."
                if 'alive' in value else f"Muestra de posición y recursos de {who}; su vida se fecha por separado.")
        if "has_c4" in value:
            text += " Es el portador del C4." if value["has_c4"] else " No porta el C4."
        resources = [f"{number(value[k])} de {label}" for k, label in
                     (("armor", "armadura"), ("money", "dinero")) if k in value]
        if resources:
            text += " Recursos de esta muestra: " + ", ".join(resources) + "."
        return text
    if key == 'own_life':
        return f"{who} está muerto, con 0 de salud según el acontecimiento de muerte registrado."
    if key == "own_inventory":
        if value["status"] == "unavailable":
            return f"Inventario de {who} no disponible."
        # Item lists are records, not an alternative authority on C4 ownership.
        equipment = [name for name in value.get("weapons", []) if name != "C4"]
        prefix = "Último inventario observado" if value["status"] == "last_observed" else "Inventario observado"
        return (f"{prefix} de {who}: {', '.join(equipment) or 'sin equipo acreditado aparte del C4'}. "
                "La posesión del C4 se describe en el estado del objetivo. "
                + ("Este inventario es histórico y no acredita equipo actual."
                   if value["status"] == "last_observed" else ""))
    if key == "round_remaining_seconds":
        return (f"Cuenta atrás de ronda: {number(value)} segundos."
                if value is not None else "No se entrega una cuenta atrás de ronda en esta muestra.")
    if key == "round_context":
        phase = value["phase"]
        if phase == "ended":
            winner = value.get("winner")
            return "Ronda terminada." + (f" Ganó {winner}." if winner else " Ganador no acreditado.")
        phase_text = {"preplant": "antes del plantado", "planted": "con el C4 plantado",
                      "freezetime": "en tiempo de preparación", "unknown": "con fase desconocida"}[phase]
        text = f"Ronda {value['round']} {phase_text}."
        clock = value.get("bomb_remaining_seconds") if phase == "planted" else value.get("round_remaining_seconds")
        if clock is not None:
            text += f" Cuenta atrás {'del C4' if phase == 'planted' else 'de ronda'}: {number(clock)} segundos."
        elif phase in ("preplant", "planted"):
            text += " Cuenta atrás no acreditada en este instante."
        if value.get("eligibility", {}).get("kind") == "unknown":
            text += " Tipo de ronda no acreditado."
        return text
    if key == "own_alive_count":
        return f"{number(value)} jugadores propios vivos en la muestra."
    if key == "place_context":
        if value["status"] == "mapped":
            return f"{who}: zona {value['zone']} según la referencia local del mapa; compatibilidad exacta con la demo no acreditada."
        return f"{who}: sin zona única acreditada en la referencia local ({value['status']})."
    if key == "enemy_contact":
        p = value.get("position", {})
        return (f"Contacto rival calculado de {who}, antigüedad {number(value.get('contact_age_seconds'))} s"
                + (f", posición ({number(p['x'])}, {number(p['y'])})" if "x" in p and "y" in p else "")
                + ". Es una estimación; no acredita visión humana real ni inventario rival.")
    if key == 'information_policy':
        return ('Los contactos se comparten inmediatamente mediante comunicación de equipo simulada. '
                'La recepción por compañeros es posible; no se acreditan conversación real, escucha, atención ni recuerdo. '
                'El estado propio/aliado es una vista colectiva de la demo y no demuestra conocimiento individual de todos sus recursos. '
                'Las etiquetas rivales enlazan registros de la demo; no acreditan reconocimiento de una persona ni una rotación percibida sólo por coincidir su ID.')
    if key == 'enemy_information':
        point = value['position']
        observers = ', '.join(alias(o['observer_id']) for o in value['observations'])
        recipients = sorted({r['player_id'] for o in value['observations'] for r in o['recipients'] if r['reception'] == 'possible'}, key=int)
        text = (f"{who}: {'última observación histórica' if value['state'] == 'last_observed' else 'contacto actual estimado'}, "
                f"hace {number(value['age_seconds'])} s, en ({number(point['x'])}, {number(point['y'])}). "
                f"Observadores registrados: {observers}.")
        place = value.get('last_observed_place', {})
        if place.get('status') == 'mapped':
            text += f" Zona de esa observación: {place['zone']} según la referencia local del mapa."
        if recipients:
            text += ' Posible recepción compartida: ' + ', '.join(alias(i) for i in recipients) + '.'
        if value['state'] == 'last_observed':
            text += ' Posición actual desconocida; esta observación no acredita que siga vivo o en ese punto.'
        return text + ' El cálculo y la comunicación simulada no acreditan percepción humana ni recepción efectiva.'
    if key == 'possible_player_information':
        if not value['observations']:
            return f'{who}: sin contactos registrados para este jugador en la simulación; otras fuentes de información no están acreditadas.'
        descriptions = []
        for observation in value['observations']:
            receipt = next(r for r in observation['recipients'] if alias(r['player_id']) == who)
            mode = ('observación individual calculada' if observation['status'] == 'estimated' else 'observación individual acreditada por la fuente') if receipt['reception'] == 'origin' else 'posible recepción de comunicación simulada'
            point = observation['position']
            descriptions.append(f"{alias(observation['subject_id'])}, {mode}, dato de hace {number(observation['age_seconds'])} s "
                                f"en ({number(point['x'])}, {number(point['y'])})")
        return f"Información disponible para {who}: " + '; '.join(descriptions) + '. Son datos fechados; no prueban conocimiento recordado ni posición actual del rival.'
    if key == 'sound_information_policy':
        return ('El alcance de los sonidos se estima con perfiles fijos del juego y distancia en tres dimensiones. '
                'La recepción directa y la comunicación inmediata al equipo son posibles. '
                'No confirma escucha, reconocimiento, número de enemigos ni posición exacta percibida. '
                'El enmascaramiento y la compatibilidad exacta con la versión de la demo siguen sin verificar.'
                if value['source_status']=='estimated_source_supplied' else 'Fuente sonora utilizable no disponible; recepción desconocida.')
    if key == 'sound_signal':
        names = {'footstep':'pasos','landing':'aterrizaje','jump':'salto','reload':'componente de recarga',
                 'scope':'zoom de mira','gunshot':'disparo','c4':'C4','grenade':'granada','knife':'cuchillo',
                 'item':'objeto','weapon_handling':'manejo del arma','world':'entorno'}
        regions = ', '.join(dict.fromkeys(c['label'] for c in value['region']['candidates'])) or 'región desconocida'
        side = value['source_side']
        return (f"Señal de {names.get(value['kind'],'sonido')}: posible recepción por {alias(value['observer_id'])}, "
                f"dato de hace {number(value['age_seconds'])} s; región compatible estimada: {regions}. "
                + (f'Origen {side} según la demo; no implica que el jugador reconociera ese bando. ' if side in ('T','CT') else 'Bando de origen desconocido. ')
                + 'No confirma escucha ni número de rivales; identidad, posición percibida exacta y posición actual desconocidas.')
    if key == 'possible_sound_information':
        return (f"{who}: {len(value['individual_signal_ids'])} señales individuales posibles y "
                f"{len(value['possible_shared_signal_ids'])} señales mediante posible comunicación de equipo retenidas. "
                'Son registros fechados, no enemigos contados ni conocimiento efectivo acreditado.')
    if key == 'utility_information_policy':
        return ('Se suministran puntos de utilidad visual estimada; la comunicación inmediata y sus receptores son posibles. '
                'No acredita percepción humana, sonido ni eficacia.' if value['source_status'] == 'estimated_source_supplied'
                else 'Fuente de observaciones visuales de utilidad no disponible.')
    if key == 'utility_signal':
        point = value['position']
        if value['position_status'] == 'ambiguous_same_tick':
            return (f"Señal estimada de {value['utility_type']}, dato de hace {number(value['age_seconds'])} s. "
                    'Los callbacks del mismo objeto/tick aportan puntos distintos: posición ambigua. '
                    'Se conserva una sola señal con todas sus fuentes, sin promediar ni inventar otra granada.')
        return (f"Señal estimada de {value['utility_type']} ({value['kind']}), dato de hace {number(value['age_seconds'])} s "
                f"en ({number(point['x'])}, {number(point['y'])}). "
                + ('Es una posición histórica; estado y posición actuales desconocidos. ' if value['state'] == 'last_observed' else 'Punto observado en este corte. ')
                + 'No identifica lanzador, origen, bando rival ni extensión completa del efecto.')
    if key == 'utility_launcher_estimate':
        return (f"Tres lanzamientos observables distintos de {value['utility_type']} en {number(value['interval_seconds'])} s: "
                'indicio estimado de al menos tres lanzadores según el criterio experto de cinco segundos. '
                'Bando, origen y posición actual desconocidos; recogidas o suministro adicional no descartados. '
                'Las flashes quedan fuera y los grupos solapados no se suman.')
    if key == 'possible_utility_information':
        return (f"{who}: {len(value['observation_ids'])} señales de utilidad retenidas con observación directa o posible comunicación simulada. "
                'No acredita recepción efectiva ni conocimiento recordado.')
    if key == 'utility_segment':
        return (f"Dos puntos de proyectil estimados por {alias(value['observer_id'])} entre ticks {value['start_tick']} y {value['end_tick']}. "
                'El tramo entre muestras es una interpolación; no acredita trayectoria anterior, origen ni posición actual del lanzador.')
    if key == "contact_status":
        labels = {"calculated_contact_visible": "Hay contactos calculados.",
                  "calculated_no_contact": "El cálculo no registra contactos.",
                  "unavailable": "Cálculo de contacto no disponible.",
                  "confirmed_contact_observed": "Hay contacto acreditado por su fuente."}
        return labels.get(value, "Estado de contacto no acreditado.") + " La ausencia de contacto no demuestra ausencia de rivales."
    if key == "own_utility_tracking":
        return (f"Seguimiento propio simulado: {len(value.get('projectiles', []))} proyectiles y "
                f"{len(value.get('active_effects', []))} efectos activos registrados. "
                "Las trayectorias muestreadas no acreditan zonas atravesadas, percepción real ni eficacia.")
    if key == "uncertainty":
        return ("No se suministran el total exacto de enemigos vivos, sus recursos ocultos ni patrones de rondas anteriores. "
                "Los contactos calculados y la comunicación de equipo simulada conservan sus límites.")
    bomb_events = {"bomb_carrier_snapshot": "Se registra el portador del C4",
                   "bomb_pickup": "Se recoge el C4", "bomb_drop": "Se suelta el C4",
                   "bomb_plant_start": "Comienza un intento de plantado",
                   "bomb_plant_abort": "Se interrumpe el intento de plantado",
                   "bomb_plant": "Se planta el C4", "bomb_defuse_start": "Comienza un intento de desactivación",
                   "bomb_defuse_abort": "Se interrumpe el intento de desactivación",
                   "bomb_defuse": "Se desactiva el C4", "bomb_explode": "Explota el C4",
                   "bomb_round_end": "Termina la ronda del objetivo"}
    if key in bomb_events:
        return bomb_events[key] + (f"; participante {who}." if fact["subject"] else ".")
    if key == "death":
        return f"Se comunica la muerte de {who}; el punto de muerte, si está disponible, es información histórica de equipo simulada."
    if key == "kill":
        killer = alias(value.get("killer_id")) if value.get("killer_id") else "causa no identificada"
        weapon = value.get('weapon') if value.get('weapon') in NATIVE_EQUIPMENT else 'no identificada en el vocabulario nativo'
        return f"Feed de muertes: {alias(value['victim_id'])} muere; autor {killer}; arma registrada {weapon}. No acredita la posición del atacante."
    if key == "weapon_fire":
        weapon = value.get('weapon') if value.get('weapon') in NATIVE_EQUIPMENT else 'un arma no identificada en el vocabulario nativo'
        return f"{alias(value.get('actor_id'))} dispara {weapon}. El disparo no acredita impacto."
    if key == "player_hurt":
        actor = f"; atacante {alias(value['actor_id'])}" if value.get("actor_id") else "; atacante no acreditado"
        return (f"{alias(value['victim_id'])}: daño reportado {number(value['damage'])}, "
                f"salud reportada después {number(value['health'])}{actor}.")
    if key in ("grenade_throw", "utility_start", "utility_end"):
        action = {"grenade_throw": "Lanzamiento", "utility_start": "Inicio de efecto", "utility_end": "Fin de efecto"}[key]
        return f"{action} de {value.get('utility_type') or 'utilidad'}; participante {alias(value.get('actor_id'))}."
    raise ValueError(f"unsupported factual description: {key}")


class StrictConsultation(comparison.Consultation):
    reference_selection = True

    def __init__(self, session):
        super().__init__(session)
        start = session._view["evolution"]["boundaries"]["start"]
        self._consulted.update(f["id"] for f in start)
        self._facts = {f["id"]: f for row in session._archive for f in row}
        self._facts.update({f["id"]: f for f in session._events})
        self._current = {f["id"] for f in session._view["situation"]}
        self._events = {f["id"] for f in session._events}

    def _fact(self, identity, section):
        require(isinstance(identity, str) and identity in self._consulted, "unconsulted or unknown fact")
        fact = self._facts[identity]
        require(section != "situation" or identity in self._current, "historical fact as current situation")
        require(section != "evolution" or identity not in self._current or identity in self._events,
                "current fact as historical evolution")
        require(section != "uncertainty" or fact["status"] in ("unknown", "estimated", "simulated"),
                "known fact cannot become unknown")
        return fact

    def _claim(self, fact):
        status = fact["status"]
        qualifier = {"observed": "Dato registrado", "estimated": "Estimación o dato histórico",
                     "simulated": "Información simulada", "unknown": "Límite de información"}[status]
        return {"kind": "observed" if status == "observed" else "limit",
                "text": f"A los {number(fact['seconds_in_window'])} s de la ventana. {qualifier}: {describe(fact)}",
                "evidence": [fact["id"]], "principles": [], "facts": [deepcopy(fact)]}

    def render(self, selection):
        require(isinstance(selection, dict) and set(selection) == set(comparison.SECTIONS), "reading sections differ")
        reading = {section: [] for section in comparison.SECTIONS}
        for section in comparison.SECTIONS:
            selected = selection[section]
            require(isinstance(selected, list) and len(selected) <= 256, "invalid reference selection")
            if section == "interpretation":
                bank = {p["id"]: p for p in comparison.BANK}
                for row in selected:
                    require(isinstance(row, dict) and set(row) == {"principle_id", "evidence"}, "free interpretation prose forbidden")
                    require(row["principle_id"] in bank and isinstance(row["evidence"], list)
                            and 0 < len(row["evidence"]) <= 32, "unsupported conditional principle")
                    facts = [self._fact(identity, section) for identity in row["evidence"]]
                    reading[section].append({"kind": "limit",
                        "text": "Criterio condicional para revisión: " + bank[row["principle_id"]]["text"]
                                + " Su selección no demuestra que se cumpla la premisa ni acredita una interpretación táctica.",
                        "evidence": list(row["evidence"]), "principles": [row["principle_id"]], "facts": deepcopy(facts)})
            else:
                require(all(isinstance(identity, str) for identity in selected), "free factual prose forbidden")
                require(len(selected) == len(set(selected)), "duplicate fact selection")
                identities = list(selected)
                for identity in self._session._view.get("required_reading_facts", {}).get(section, []):
                    if identity not in identities:
                        identities.append(identity)
                reading[section] = [self._claim(self._fact(identity, section)) for identity in identities]
        require(any(reading.values()), "empty reading")
        verification = {"citation_integrity": True, "claims": sum(map(len, reading.values())),
                        "factual_integrity": "checked_reference_rendering", "representation": VERSION,
                        "semantic_truth": "not_evaluated", "tactical_quality": "not_evaluated",
                        "queries": deepcopy(self._queries), "response_bytes": self._bytes}
        return reading, verification

    def verify(self, reading):
        require(isinstance(reading, dict) and set(reading) == set(comparison.SECTIONS), "reading sections differ")
        selection = {section: [] for section in comparison.SECTIONS}
        for section, claims in reading.items():
            require(isinstance(claims, list), "claims must be arrays")
            for claim in claims:
                require(isinstance(claim, dict) and set(claim) == {"kind", "text", "evidence", "principles", "facts"},
                        "rendered claim fields differ")
                require(isinstance(claim["evidence"], list) and claim["evidence"], "missing rendered evidence")
                if section == "interpretation":
                    require(isinstance(claim["principles"], list) and len(claim["principles"]) == 1, "invalid principle")
                    selection[section].append({"principle_id": claim["principles"][0], "evidence": claim["evidence"]})
                else:
                    require(len(claim["evidence"]) == 1 and not claim["principles"], "invalid factual citation")
                    selection[section].append(claim["evidence"][0])
        expected, verification = self.render(selection)
        require(reading == expected, "rendered fact or text differs from evidence")
        return verification

