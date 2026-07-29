import React from "react";
import * as GQL from "src/core/generated-graphql";
import { AudioMarkerCard } from "./AudioMarkerCard";

interface IAudioMarkerCardGridProps {
  markers: GQL.AudioMarkerDataFragment[];
  selectedIds?: Set<string>;
  zoomIndex?: number;
  onSelectChange?: (id: string, selected: boolean, shiftKey: boolean) => void;
}

export const AudioMarkerCardGrid: React.FC<IAudioMarkerCardGridProps> = ({
  markers,
  selectedIds,
  zoomIndex,
  onSelectChange,
}) => {
  return (
    <div className="audio-marker-card-grid">
      {markers.map((marker, index) => (
        <AudioMarkerCard
          key={marker.id}
          marker={marker}
          index={index}
          zoomIndex={zoomIndex}
          selecting={selectedIds !== undefined}
          selected={selectedIds?.has(marker.id)}
          onSelectedChanged={(selected, shiftKey) =>
            onSelectChange?.(marker.id, selected, shiftKey)
          }
        />
      ))}
    </div>
  );
};
