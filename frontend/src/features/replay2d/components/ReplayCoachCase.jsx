import { useMemo, useState } from "react";
import Replay2DViewerV2 from "./Replay2DViewerV2";
import { ReplayCoachReading } from "./ReplayCoachReading";
import { createTeamPlayerLabels } from "../domain/replayPlayerLabels";

// Mount with a case_id key when changing cases, so playback and drafts stay separate.
export function ReplayCoachCase({ data, bundle, mapImageSources }) {
  const replayData = useMemo(() => ({ ...data, rounds: data.rounds.filter((row) => row.round === bundle.round) }), [data, bundle.round]);
  const playerLabels = useMemo(() => createTeamPlayerLabels(replayData.rounds[0]?.frames[0]?.players), [replayData]);
  const reviewBundle = useMemo(() => ({ ...bundle, review_supplement: { ...bundle.review_supplement,
    display_player_labels: playerLabels, full_round_available: true,
    post_cut_information_not_given_to_model: true,
  } }), [bundle, playerLabels]);
  const [context, setContext] = useState(null);
  const [control, setControl] = useState({ round: bundle.round, tick: bundle.replay_cut_tick, play: false });
  return <div className="demo-review-layout">
    <Replay2DViewerV2 replayData={replayData} initialRound={bundle.round} initialPerspective="T"
      mapImageSources={mapImageSources} playerLabels={playerLabels} externalControl={control} onPositionChange={setContext} />
    <ReplayCoachReading bundle={reviewBundle} context={context} playerLabels={playerLabels}
      onGoToCut={() => setControl({ round: bundle.round, tick: bundle.replay_cut_tick, play: false })}
      onPlayFromStart={() => setControl({ round: bundle.round, tick: bundle.start_tick, play: true })} />
  </div>;
}
