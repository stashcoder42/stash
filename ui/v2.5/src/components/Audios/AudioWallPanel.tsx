import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Form } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import { AudioQueue } from "src/models/audioQueue";
import Gallery, {
  GalleryI,
  PhotoProps,
  RenderImageProps,
} from "react-photo-gallery";
import { useConfigurationContext } from "src/hooks/Config";
import { objectTitle } from "src/core/files";
import { Link, useHistory } from "react-router-dom";
import { TruncatedText } from "../Shared/TruncatedText";
import TextUtils from "src/utils/text";
import { useIntl } from "react-intl";
import { useDragMoveSelect } from "../Shared/GridCard/dragMoveSelect";
import cx from "classnames";

interface IAudioPhoto {
  audio: GQL.SlimAudioDataFragment;
  link: string;
  onError?: (photo: PhotoProps<IAudioPhoto>) => void;
}

interface IExtraProps {
  maxHeight: number;
  selected?: boolean;
  onSelectedChanged?: (selected: boolean, shiftKey: boolean) => void;
  selecting?: boolean;
}

export const AudioWallItem: React.FC<
  RenderImageProps<IAudioPhoto> & IExtraProps
> = (props: RenderImageProps<IAudioPhoto> & IExtraProps) => {
  const intl = useIntl();

  const { dragProps } = useDragMoveSelect({
    selecting: props.selecting || false,
    selected: props.selected || false,
    onSelectedChanged: props.onSelectedChanged,
  });

  const { configuration } = useConfigurationContext();
  const showTitle = configuration?.interface.wallShowTitle ?? false;

  const height = Math.min(props.maxHeight, props.photo.height);
  const zoomFactor = height / props.photo.height;
  const width = props.photo.width * zoomFactor;

  type style = Record<string, string | number | undefined>;
  var divStyle: style = {
    margin: props.margin,
    display: "block",
  };

  if (props.direction === "column") {
    divStyle.position = "absolute";
    divStyle.left = props.left;
    divStyle.top = props.top;
  }

  var handleClick = function handleClick(event: React.MouseEvent) {
    if (props.selecting && props.onSelectedChanged) {
      props.onSelectedChanged(!props.selected, event.shiftKey);
      event.preventDefault();
      event.stopPropagation();
      return;
    }
    if (props.onClick) {
      props.onClick(event, { index: props.index });
    }
  };

  const { audio } = props.photo;
  const title = objectTitle(audio);
  const performerNames = audio.performers.map((p) => p.name);
  const performers =
    performerNames.length >= 2
      ? [...performerNames.slice(0, -2), performerNames.slice(-2).join(" & ")]
      : performerNames;

  let shiftKey = false;

  return (
    <div
      className={cx("wall-item", { "show-title": showTitle })}
      role="button"
      onClick={handleClick}
      {...dragProps}
      style={{
        ...divStyle,
        width,
        height,
      }}
    >
      {props.onSelectedChanged && (
        <Form.Control
          type="checkbox"
          className="wall-item-check mousetrap"
          checked={props.selected}
          onChange={() => props.onSelectedChanged!(!props.selected, shiftKey)}
          onClick={(event: React.MouseEvent<HTMLInputElement, MouseEvent>) => {
            shiftKey = event.shiftKey;
            event.stopPropagation();
          }}
        />
      )}
      <img
        loading="lazy"
        key={props.photo.key}
        src={props.photo.src}
        width={width}
        height={height}
        alt={props.photo.alt}
        onClick={handleClick}
        onError={() => {
          props.photo.onError?.(props.photo);
        }}
      />
      <div className="lineargradient">
        <footer className="wall-item-footer">
          <Link to={props.photo.link} onClick={(e) => e.stopPropagation()}>
            {title && (
              <TruncatedText
                text={title}
                lineCount={1}
                className="wall-item-title"
              />
            )}
            <TruncatedText text={performers.join(", ")} />
            <div>
              {audio.date && TextUtils.formatFuzzyDate(intl, audio.date)}
            </div>
          </Link>
        </footer>
      </div>
    </div>
  );
};

function getDimensions(a: GQL.SlimAudioDataFragment) {
  // Audio uses square dimensions for cover art
  const defaults = { width: 300, height: 300 };
  return defaults;
}

interface IAudioWallProps {
  audios: GQL.SlimAudioDataFragment[];
  audioQueue?: AudioQueue;
  zoomIndex: number;
  selectedIds?: Set<string>;
  onSelectChange?: (id: string, selected: boolean, shiftKey: boolean) => void;
  selecting?: boolean;
}

// HACK: typescript doesn't allow Gallery to accept a parameter for some reason
const AudioGallery = Gallery as unknown as GalleryI<IAudioPhoto>;

const breakpointZoomHeights = [
  { minWidth: 576, heights: [100, 120, 240, 360] },
  { minWidth: 768, heights: [120, 160, 240, 480] },
  { minWidth: 1200, heights: [120, 160, 240, 300] },
  { minWidth: 1400, heights: [160, 240, 300, 480] },
];

const AudioWall: React.FC<IAudioWallProps> = ({
  audios,
  audioQueue,
  zoomIndex,
  selectedIds,
  onSelectChange,
  selecting,
}) => {
  const history = useHistory();

  const containerRef = React.useRef<HTMLDivElement>(null);

  const margin = 3;
  const direction = "row";

  const [erroredImgs, setErroredImgs] = useState<string[]>([]);

  const handleError = useCallback((photo: PhotoProps<IAudioPhoto>) => {
    setErroredImgs((prev) => [...prev, photo.src]);
  }, []);

  useEffect(() => {
    setErroredImgs([]);
  }, [audios]);

  const photos: PhotoProps<IAudioPhoto>[] = useMemo(() => {
    return audios.map((a, index) => {
      const { width, height } = getDimensions(a);

      return {
        audio: a,
        src:
          a.paths.cover && !erroredImgs.includes(a.paths.cover)
            ? a.paths.cover!
            : a.paths.cover!,
        link: audioQueue
          ? audioQueue.makeLink(a.id, { audioIndex: index })
          : `/audios/${a.id}`,
        width,
        height,
        tabIndex: index,
        key: a.id,
        loading: "lazy",
        alt: objectTitle(a),
        onError: handleError,
      };
    });
  }, [audios, audioQueue, erroredImgs, handleError]);

  const onClick = useCallback(
    (event, { index }) => {
      history.push(photos[index].link);
    },
    [history, photos]
  );

  function columns(containerWidth: number) {
    const preferredSize = 300;
    const columnCount = containerWidth / preferredSize;
    return Math.round(columnCount);
  }

  const targetRowHeight = useCallback(
    (containerWidth: number) => {
      let zoomHeight = 280;
      breakpointZoomHeights.forEach((e) => {
        if (containerWidth >= e.minWidth) {
          zoomHeight = e.heights[zoomIndex];
        }
      });
      return zoomHeight;
    },
    [zoomIndex]
  );

  // set the max height as a factor of the targetRowHeight
  // this allows some images to be taller than the target row height
  // but prevents images from becoming too tall when there is a small number of items
  const maxHeightFactor = 1.3;

  const renderImage = useCallback(
    (props: RenderImageProps<IAudioPhoto>) => {
      const audioId = props.photo.audio.id;
      return (
        <AudioWallItem
          {...props}
          maxHeight={
            targetRowHeight(containerRef.current?.offsetWidth ?? 0) *
            maxHeightFactor
          }
          selected={selectedIds?.has(audioId)}
          onSelectedChanged={
            onSelectChange
              ? (selected, shiftKey) =>
                  onSelectChange(audioId, selected, shiftKey)
              : undefined
          }
          selecting={selecting}
        />
      );
    },
    [targetRowHeight, selectedIds, onSelectChange, selecting]
  );

  return (
    <div className={`audio-wall`} ref={containerRef}>
      {photos.length ? (
        <AudioGallery
          photos={photos}
          renderImage={renderImage}
          onClick={onClick}
          margin={margin}
          direction={direction}
          columns={columns}
          targetRowHeight={targetRowHeight}
        />
      ) : null}
    </div>
  );
};

interface IAudioWallPanelProps {
  audios: GQL.SlimAudioDataFragment[];
  audioQueue?: AudioQueue;
  zoomIndex: number;
  selectedIds?: Set<string>;
  onSelectChange?: (id: string, selected: boolean, shiftKey: boolean) => void;
}

export const AudioWallPanel: React.FC<IAudioWallPanelProps> = ({
  audios,
  audioQueue,
  zoomIndex,
  selectedIds,
  onSelectChange,
}) => {
  const selecting = !!selectedIds && selectedIds.size > 0;
  return (
    <AudioWall
      audios={audios}
      audioQueue={audioQueue}
      zoomIndex={zoomIndex}
      selectedIds={selectedIds}
      onSelectChange={onSelectChange}
      selecting={selecting}
    />
  );
};
