import React from "react";
import { Form } from "react-bootstrap";
import { Link } from "react-router-dom";
import cx from "classnames";
import * as GQL from "src/core/generated-graphql";
import { markerTitle } from "src/core/markers";
import TextUtils from "src/utils/text";
import { useConfigurationContext } from "src/hooks/Config";
import { TruncatedText } from "../Shared/TruncatedText";

interface IAudioMarkerWallPanelProps {
  markers: GQL.AudioMarkerDataFragment[];
  clickHandler?: (
    e: React.MouseEvent,
    marker: GQL.AudioMarkerDataFragment
  ) => void;
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

function wallItemTitle(marker: GQL.AudioMarkerDataFragment) {
  const newTitle = markerTitle(marker);
  const seconds = TextUtils.formatTimestampRange(
    marker.seconds,
    marker.end_seconds ?? undefined
  );
  if (newTitle) {
    return `${newTitle} - ${seconds}`;
  }
  return seconds;
}

interface IAudioMarkerWallItemProps {
  marker: GQL.AudioMarkerDataFragment;
  className?: string;
  onClick?: (e: React.MouseEvent) => void;
  selecting?: boolean;
  selected?: boolean;
  onSelectedChanged?: (selected: boolean, shiftKey: boolean) => void;
}

// Audio markers have no animated/video preview, so - unlike scene's
// MarkerWallItem - this always renders an <img>. There is deliberately no
// react-photo-gallery here either: the audio marker panel is a plain flex row
// and audio's wall display mode is a non-goal.
// See docs/AUDIO_SCENE_PARITY.md.
export const AudioMarkerWallItem: React.FC<IAudioMarkerWallItemProps> = ({
  marker,
  className,
  onClick,
  selecting,
  selected,
  onSelectedChanged,
}) => {
  const { configuration } = useConfigurationContext();
  const showTitle = configuration?.interface.wallShowTitle ?? false;

  // For audio markers, we use the audio's cover image as the preview.
  const imageSrc = marker.preview || marker.audio.paths.cover;

  const title = wallItemTitle(marker);
  const tagNames = marker.tags.map((t) => t.name);
  const link = `/audios/${marker.audio.id}?t=${marker.seconds}`;

  let shiftKey = false;

  function handleClick(event: React.MouseEvent) {
    if (selecting && onSelectedChanged) {
      onSelectedChanged(!selected, event.shiftKey);
      event.preventDefault();
      event.stopPropagation();
      return;
    }
    onClick?.(event);
  }

  return (
    <div
      className={cx("wall-item", className, { "show-title": showTitle })}
      role="button"
      onClick={handleClick}
    >
      {onSelectedChanged && (
        <Form.Control
          type="checkbox"
          className="wall-item-check mousetrap"
          checked={selected ?? false}
          onChange={() => onSelectedChanged(!selected, shiftKey)}
          onClick={(event: React.MouseEvent<HTMLInputElement, MouseEvent>) => {
            shiftKey = event.shiftKey;
            event.stopPropagation();
          }}
        />
      )}
      <img
        loading="lazy"
        className="wall-item-media"
        src={imageSrc ?? undefined}
        alt={title}
      />
      <div className="lineargradient">
        <footer className="wall-item-footer">
          <Link to={link} onClick={(e) => e.stopPropagation()}>
            {title && (
              <TruncatedText
                text={title}
                lineCount={1}
                className="wall-item-title"
              />
            )}
            <TruncatedText text={tagNames.join(", ")} />
          </Link>
        </footer>
      </div>
    </div>
  );
};

export const AudioMarkerWallPanel: React.FC<IAudioMarkerWallPanelProps> = ({
  markers,
  clickHandler,
  zoomIndex,
  selectedIds,
  onSelectChange,
}) => {
  const selecting = !!selectedIds && selectedIds.size > 0;
  return (
    <div className="row">
      <div
        className={cx(
          "wall w-100 row justify-content-center audio-marker-wall",
          zoomIndex !== undefined ? `zoom-${zoomIndex}` : undefined
        )}
      >
        {markers.map((marker, index) => (
          <AudioMarkerWallItem
            key={marker.id}
            marker={marker}
            className={calculateClass(index, markers.length)}
            onClick={(e) => clickHandler?.(e, marker)}
            selecting={selecting}
            selected={selectedIds?.has(marker.id)}
            onSelectedChanged={
              onSelectChange
                ? (selected, shiftKey) =>
                    onSelectChange(marker.id, selected, shiftKey)
                : undefined
            }
          />
        ))}
      </div>
    </div>
  );
};
