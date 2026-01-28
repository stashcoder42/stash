import React, { useMemo } from "react";
import { Button, ButtonGroup, OverlayTrigger, Tooltip } from "react-bootstrap";
import cx from "classnames";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "../Shared/Icon";
import { TagLink } from "../Shared/TagLink";
import { HoverPopover } from "../Shared/HoverPopover";
import { TruncatedText } from "../Shared/TruncatedText";
import TextUtils from "src/utils/text";
import { AudioQueue } from "src/models/audioQueue";
import { PerformerPopoverButton } from "../Shared/PerformerPopoverButton";
import { GridCard } from "../Shared/GridCard/GridCard";
import { RatingBanner } from "../Shared/RatingBanner";
import {
  faBox,
  faHeadphones,
  faTag,
} from "@fortawesome/free-solid-svg-icons";
import { objectPath, objectTitle } from "src/core/files";
import { PatchComponent } from "src/patch";
import { OCounterButton } from "../Shared/CountButton";

interface IAudioPreviewProps {
  image?: string;
}

export const AudioPreview: React.FC<IAudioPreviewProps> = ({ image }) => {
  return (
    <div className={cx("audio-card-preview")}>
      {image ? (
        <img
          className="audio-card-preview-image"
          loading="lazy"
          src={image}
          alt=""
        />
      ) : (
        <div className="audio-card-preview-placeholder">
          <Icon icon={faHeadphones} size="3x" />
        </div>
      )}
    </div>
  );
};

interface IAudioCardProps {
  audio: GQL.SlimAudioDataFragment;
  width?: number;
  index?: number;
  queue?: AudioQueue;
  compact?: boolean;
  selecting?: boolean;
  selected?: boolean | undefined;
  zoomIndex?: number;
  onSelectedChanged?: (selected: boolean, shiftKey: boolean) => void;
}

const AudioCardPopovers = PatchComponent(
  "AudioCard.Popovers",
  (props: IAudioCardProps) => {
    function maybeRenderTagPopoverButton() {
      if (props.audio.tags.length <= 0) return;

      const popoverContent = props.audio.tags.map((tag) => (
        <TagLink key={tag.id} tag={tag} />
      ));

      return (
        <HoverPopover
          className="tag-count"
          placement="bottom"
          content={popoverContent}
        >
          <Button className="minimal">
            <Icon icon={faTag} />
            <span>{props.audio.tags.length}</span>
          </Button>
        </HoverPopover>
      );
    }

    function maybeRenderPerformerPopoverButton() {
      if (props.audio.performers.length <= 0) return;

      return (
        <PerformerPopoverButton
          performers={props.audio.performers}
          linkType="audio"
        />
      );
    }

    function maybeRenderOCounter() {
      if (props.audio.o_counter) {
        return <OCounterButton value={props.audio.o_counter} />;
      }
    }

    function maybeRenderOrganized() {
      if (props.audio.organized) {
        return (
          <OverlayTrigger
            overlay={<Tooltip id="organised-tooltip">{"Organized"}</Tooltip>}
            placement="bottom"
          >
            <div className="organized">
              <Button className="minimal">
                <Icon icon={faBox} />
              </Button>
            </div>
          </OverlayTrigger>
        );
      }
    }

    function maybeRenderPopoverButtonGroup() {
      if (
        !props.compact &&
        (props.audio.tags.length > 0 ||
          props.audio.performers.length > 0 ||
          props.audio?.o_counter ||
          props.audio.organized)
      ) {
        return (
          <>
            <hr />
            <ButtonGroup className="card-popovers">
              {maybeRenderTagPopoverButton()}
              {maybeRenderPerformerPopoverButton()}
              {maybeRenderOCounter()}
              {maybeRenderOrganized()}
            </ButtonGroup>
          </>
        );
      }
    }

    return <>{maybeRenderPopoverButtonGroup()}</>;
  }
);

const AudioCardDetails = PatchComponent(
  "AudioCard.Details",
  (props: IAudioCardProps) => {
    return (
      <div className="scene-card__details">
        <span className="scene-card__date">{props.audio.date}</span>
        <span className="file-path extra-scene-info">
          {objectPath(props.audio)}
        </span>
        <TruncatedText
          className="scene-card__description"
          text={props.audio.details}
          lineCount={3}
        />
      </div>
    );
  }
);

const AudioCardOverlays = PatchComponent(
  "AudioCard.Overlays",
  (props: IAudioCardProps) => {
    // Audio cards don't have studio overlays like scenes
    return null;
  }
);

const AudioCardImage = PatchComponent(
  "AudioCard.Image",
  (props: IAudioCardProps) => {
    const file = useMemo(
      () => (props.audio.files.length > 0 ? props.audio.files[0] : undefined),
      [props.audio]
    );

    function maybeRenderAudioSpecsOverlay() {
      return (
        <div className="audio-specs-overlay">
          {(file?.duration ?? 0) >= 1 ? (
            <span className="overlay-duration">
              {TextUtils.secondsToTimestamp(file?.duration ?? 0)}
            </span>
          ) : (
            ""
          )}
        </div>
      );
    }

    return (
      <>
        <AudioPreview image={props.audio.paths.cover ?? undefined} />
        <RatingBanner rating={props.audio.rating100} />
        {maybeRenderAudioSpecsOverlay()}
      </>
    );
  }
);

export const AudioCard = PatchComponent(
  "AudioCard",
  (props: IAudioCardProps) => {
    const file = useMemo(
      () => (props.audio.files.length > 0 ? props.audio.files[0] : undefined),
      [props.audio]
    );

    function zoomIndex() {
      if (!props.compact && props.zoomIndex !== undefined) {
        return `zoom-${props.zoomIndex}`;
      }

      return "";
    }

    function filelessClass() {
      if (!props.audio.files.length) {
        return "fileless";
      }

      return "";
    }

    const audioLink = props.queue
      ? props.queue.makeLink(props.audio.id, { audioIndex: props.index })
      : `/audios/${props.audio.id}`;

    return (
      <GridCard
        className={`audio-card ${zoomIndex()} ${filelessClass()}`}
        url={audioLink}
        title={objectTitle(props.audio)}
        width={props.width}
        linkClassName="audio-card-link"
        thumbnailSectionClassName="audio-section"
        resumeTime={props.audio.resume_time ?? undefined}
        duration={file?.duration ?? undefined}
        image={<AudioCardImage {...props} />}
        overlays={<AudioCardOverlays {...props} />}
        details={<AudioCardDetails {...props} />}
        popovers={<AudioCardPopovers {...props} />}
        selected={props.selected}
        selecting={props.selecting}
        onSelectedChanged={props.onSelectedChanged}
      />
    );
  }
);
