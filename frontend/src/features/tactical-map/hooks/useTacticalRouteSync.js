import { useEffect, useRef } from 'react';
import { parseTacticalRouteState } from '../../../context/TacticalMapContext';

const canonicalizeSignalId = (signalId) => {
  if (!signalId) return null;
  return signalId
    .replace(/^territory(?=:|$)/, 'habit')
    .replace(/^impact(?=:|$)/, 'strength');
};

export const useTacticalRouteSync = ({
  searchParams,
  setSearchParams,
  dispatch,
  routeState,
  selectedSignalId,
  setSelectedSignalId,
  hasLevels,
  deferRouteWrite = false,
}) => {
  const suppressWriteRef = useRef(false);
  const currentStateRef = useRef(null);
  currentStateRef.current = { ...routeState, selectedSignalId };
  const searchSignature = searchParams.toString();

  useEffect(() => {
    const params = new URLSearchParams(searchSignature);
    const requestedRoute = parseTacticalRouteState(params);
    const requestedSignal = requestedRoute.activeSection === 'briefing'
      && /^(strength|habit|risk|territory|impact)(:|$)/.test(params.get('signal') || '')
      ? canonicalizeSignalId(params.get('signal'))
      : null;
    const current = currentStateRef.current;
    const routeChanged = Object.entries(requestedRoute).some(([key, value]) => current?.[key] !== value);
    const signalChanged = current?.selectedSignalId !== requestedSignal;
    if (!routeChanged && !signalChanged) return;

    suppressWriteRef.current = true;
    if (routeChanged) dispatch({ type: 'SYNC_ROUTE_STATE', payload: requestedRoute });
    if (signalChanged) setSelectedSignalId(requestedSignal);
  }, [dispatch, searchSignature, setSelectedSignalId]);

  useEffect(() => {
    if (suppressWriteRef.current) {
      suppressWriteRef.current = false;
      return;
    }
    if (deferRouteWrite) return;

    const nextParams = new URLSearchParams(searchParams);
    nextParams.set('map', routeState.currentMap);
    nextParams.set('lens', routeState.activeSection);
    nextParams.set('side', routeState.activeSide);
    if (hasLevels) nextParams.set('level', routeState.currentLevel);
    else nextParams.delete('level');
    if (routeState.selectedCallout) nextParams.set('zone', routeState.selectedCallout);
    else nextParams.delete('zone');
    if (routeState.activeSection === 'combat') nextParams.set('metric', routeState.combatMetric);
    else nextParams.delete('metric');
    if (selectedSignalId && routeState.activeSection === 'briefing') nextParams.set('signal', selectedSignalId);
    else nextParams.delete('signal');

    if (nextParams.toString() !== searchParams.toString()) {
      setSearchParams(nextParams, { replace: true });
    }
  }, [deferRouteWrite, hasLevels, routeState, searchParams, selectedSignalId, setSearchParams]);
};
