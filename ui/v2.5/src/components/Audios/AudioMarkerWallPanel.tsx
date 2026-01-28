import React, { MouseEvent } from "react";
import * as GQL from "src/core/generated-graphql";
import { AudioMarkerCard } from "./AudioMarkerCard";

interface IAudioMarkerWallPanelProps {
  markers: GQL.AudioMarkerDataFragment[];
  clickHandler?: (e: MouseEvent, marker: GQL.AudioMarkerDataFragment) => void;
  zoomIndex?: number;
  selectedIds?: Set<string>;
  onSelectChange?: (id: string, selected: boolean, shiftKey: boolean) => void;
}

const calculateClass = (index: number, count: number) => {
  // First position and more than one row
  if (index === 0 && count > 5) return "transform-origin-top-left";
  // Fifth position and more than one row
  if (index === 4 && count > 5) return "transform-origin-top-right";
  // Top row
  if (index < 5) return "transform-origin-top";
  // Two or more rows, with full last row and index is last
  if (count > 9 && count % 5 === 0 && index + 1 === count)
    return "transform-origin-bottom-right";
  // Two or more rows, with full last row and index is fifth to last
  if (count > 9 && count % 5 === 0 && index + 5 === count)
    return "transform-origin-bottom-left";
  // Multiple of five minus one
  if (index % 5 === 4) return "transform-origin-right";
  // Multiple of five
  if (index % 5 === 0) return "transform-origin-left";
  // Position is equal or larger than first position in last row
  if (count - (count % 5 || 5) <= index + 1) return "transform-origin-bottom";
  // Default
  return "transform-origin-center";
};

export const AudioMarkerWallPanel: React.FC<IAudioMarkerWallPanelProps> = ({
  markers,
  clickHandler,
  zoomIndex,
  selectedIds,
  onSelectChange,
}) => {
  return (
    <div className="row">
      <div className="wall w-100 row justify-content-center audio-marker-wall">
        {markers.map((marker, index) => (
          <div
            key={marker.id}
            className={`wall-item ${calculateClass(index, markers.length)}`}
            onClick={(e) => clickHandler?.(e, marker)}
          >
            <AudioMarkerCard
              marker={marker}
              index={index}
              zoomIndex={zoomIndex}
              selecting={selectedIds !== undefined}
              selected={selectedIds?.has(marker.id)}
              onSelectedChanged={(selected, shiftKey) =>
                onSelectChange?.(marker.id, selected, shiftKey)
              }
            />
          </div>
        ))}
      </div>
    </div>
  );
};
