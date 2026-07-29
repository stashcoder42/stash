import React, { useState, useEffect } from "react";
import { Button } from "react-bootstrap";
import { FormattedMessage } from "react-intl";
import Mousetrap from "mousetrap";
import * as GQL from "src/core/generated-graphql";
import { PrimaryTags } from "./PrimaryTags";
import { AudioMarkerForm } from "./AudioMarkerForm";
import { AudioMarkerWallPanel } from "../AudioMarkerWallPanel";

interface IAudioMarkersPanelProps {
  audioId: string;
  isVisible: boolean;
  onClickMarker: (marker: GQL.AudioMarkerDataFragment) => void;
}

export const AudioMarkersPanel: React.FC<IAudioMarkersPanelProps> = ({
  audioId,
  isVisible,
  onClickMarker,
}) => {
  const { data, loading } = GQL.useFindAudioMarkerTagsQuery({
    variables: { id: audioId },
  });
  const [isEditorOpen, setIsEditorOpen] = useState<boolean>(false);
  const [editingMarker, setEditingMarker] =
    useState<GQL.AudioMarkerDataFragment>();

  // set up hotkeys
  useEffect(() => {
    if (!isVisible) return;

    Mousetrap.bind("n", () => onOpenEditor());

    return () => {
      Mousetrap.unbind("n");
    };
  });

  if (loading) return null;

  function onOpenEditor(marker?: GQL.AudioMarkerDataFragment) {
    setIsEditorOpen(true);
    setEditingMarker(marker ?? undefined);
  }

  const closeEditor = () => {
    setEditingMarker(undefined);
    setIsEditorOpen(false);
  };

  if (isEditorOpen)
    return (
      <AudioMarkerForm
        audioID={audioId}
        marker={editingMarker}
        onClose={closeEditor}
      />
    );

  // Extract all audio markers from the organized-by-tag structure
  const audioMarkers =
    data?.audioMarkerTags?.flatMap((tagGroup) => tagGroup.audio_markers) ?? [];

  return (
    <div className="audio-markers-panel">
      <Button onClick={() => onOpenEditor()}>
        <FormattedMessage id="actions.create_marker" />
      </Button>
      <div className="container">
        <PrimaryTags
          audioMarkers={audioMarkers}
          onClickMarker={onClickMarker}
          onEdit={onOpenEditor}
        />
      </div>
      <AudioMarkerWallPanel
        markers={audioMarkers}
        clickHandler={(e, marker) => {
          e.preventDefault();
          window.scrollTo(0, 0);
          onClickMarker(marker);
        }}
      />
    </div>
  );
};

export default AudioMarkersPanel;
