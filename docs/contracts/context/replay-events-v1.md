# Acontecimientos causales del replay

Extensión candidata `stratai.replay_events@1`, proyección de equipo @7 y entrada de
interpretación @5. Optativa bajo CaptureCalculatedVision; no altera los 116 schemas
del bundle ni habilita capacidades de producto. @1–@4 conservan su semántica.

Cada callback conserva tipo, tick de disponibilidad, participantes disponibles y
referencia. Se registran WeaponFire, PlayerHurt, GrenadeProjectileThrow e inicio/final
de efectos. Los disparos de cuchillo/utilidad no se presentan como tiros de arma de
fuego. Vida cero y coordenada cero observadas se distinguen de ausencia.

La perspectiva incluye acciones propias y acciones enemigas con contacto calculado
en el instante del evento, vigente menos de cuatro ticks, observador vivo y política
calculada habilitada. La estimación no acredita visión real ni audición. El daño
incluye únicamente víctimas propias; se oculta atacante y arma cuando no se conoce
al atacante. La salud enemiga queda fuera. Un contacto posterior no recupera
retroactivamente una acción oculta.

Los lanzamientos tienen identidad local. El enlace de humo con lanzamiento usa la
identidad nativa del equipamiento; el inferno conserva autor nativo e identidad de
efecto separados si no hay enlace positivo con proyectil. No se sustituye por una
correlación de cercanía. Trayectoria y geometría propias se etiquetan como seguimiento
de equipo simulado, no percepción individual acreditada ni comunicación real.
Un efecto iniciado termina la representación del proyectil enlazado; un fin observado
termina el efecto incluso entre frames. No se infieren expiración ni cuenta atrás.

El feed de muertes conserva identidades públicas, arma y tiempo sin posición o
recursos del atacante. Los cambios del C4 mantienen su contrato anterior. La secuencia
permite examinar posibles intercambios, sin clasificar trade, apoyo eficaz o intención
sólo por proximidad temporal o espacial.

Se excluyen resultados finales de disparos, daño/víctimas agregados de granada,
duraciones finales, trayectoria futura, geometría de utilidad enemiga sin observación
y fuentes sonoras no disponibles. Cada ausencia sigue explícita. Mapa y modelo usan
la misma proyección causal; versiones originales, lecturas y notas permanecen intactas.

Verificación: pruebas por capas, revisión independiente y una demo aprobada en TEMP
nuevo. La comparación local posterior al par ronda + acontecimientos conserva modelo,
configuración, cortes y errores. Es revisión asistida, no evaluación independiente.
