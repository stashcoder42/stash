import { Button, ButtonGroup } from "react-bootstrap";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "../Shared/Icon";
import { TagLink } from "../Shared/TagLink";
import { HoverPopover } from "../Shared/HoverPopover";
import TextUtils from "src/utils/text";
import { GridCard } from "../Shared/GridCard/GridCard";
import { faTag } from "@fortawesome/free-solid-svg-icons";
import { markerTitle } from "src/core/markers";
import { Link } from "react-router-dom";
import { PatchComponent } from "src/patch";
import { TruncatedText } from "../Shared/TruncatedText";

interface IAudioMarkerCardProps {
  marker: GQL.AudioMarkerDataFragment;
  cardWidth?: number;
  previewHeight?: number;
  index?: number;
  compact?: boolean;
  selecting?: boolean;
  selected?: boolean | undefined;
  zoomIndex?: number;
  onSelectedChanged?: (selected: boolean, shiftKey: boolean) => void;
}

const AudioMarkerCardPopovers = PatchComponent(
  "AudioMarkerCard.Popovers",
  (props: IAudioMarkerCardProps) => {
    function renderTagPopoverButton() {
      const popoverContent = [
        <TagLink
          key={props.marker.primary_tag.id}
          tag={props.marker.primary_tag}
          linkType="audio_marker"
        />,
      ];

      props.marker.tags.map((tag) =>
        popoverContent.push(
          <TagLink key={tag.id} tag={tag} linkType="audio_marker" />
        )
      );

      return (
        <HoverPopover
          className="tag-count"
          placement="bottom"
          content={popoverContent}
        >
          <Button className="minimal">
            <Icon icon={faTag} />
            <span>{popoverContent.length}</span>
          </Button>
        </HoverPopover>
      );
    }

    function renderPopoverButtonGroup() {
      if (!props.compact) {
        return (
          <>
            <hr />
            <ButtonGroup className="card-popovers">
              {renderTagPopoverButton()}
            </ButtonGroup>
          </>
        );
      }
    }

    return <>{renderPopoverButtonGroup()}</>;
  }
);

const AudioMarkerCardDetails = PatchComponent(
  "AudioMarkerCard.Details",
  (props: IAudioMarkerCardProps) => {
    return (
      <div className="audio-marker-card__details">
        <span className="audio-marker-card__time">
          {TextUtils.formatTimestampRange(
            props.marker.seconds,
            props.marker.end_seconds ?? undefined
          )}
        </span>
        <TruncatedText
          className="audio-marker-card__audio"
          lineCount={3}
          text={
            <Link to={`/audios/${props.marker.audio.id}`}>
              {props.marker.audio.title || "Untitled Audio"}
            </Link>
          }
        />
      </div>
    );
  }
);

const AudioMarkerCardImage = PatchComponent(
  "AudioMarkerCard.Image",
  (props: IAudioMarkerCardProps) => {
    // For audio markers, we use the audio's cover image as the preview
    const imageSrc = props.marker.preview || props.marker.audio.paths.cover;

    function maybeRenderDurationOverlay() {
      return (
        <div className="audio-specs-overlay">
          {props.marker.end_seconds && (
            <span className="overlay-duration">
              {TextUtils.secondsToTimestamp(
                props.marker.end_seconds - props.marker.seconds
              )}
            </span>
          )}
        </div>
      );
    }

    return (
      <>
        <img
          loading="lazy"
          className="audio-marker-card-image"
          alt={markerTitle(props.marker)}
          src={imageSrc ?? undefined}
        />
        {maybeRenderDurationOverlay()}
      </>
    );
  }
);

export const AudioMarkerCard = PatchComponent(
  "AudioMarkerCard",
  (props: IAudioMarkerCardProps) => {
    function zoomIndex() {
      if (!props.compact && props.zoomIndex !== undefined) {
        return `zoom-${props.zoomIndex}`;
      }

      return "";
    }

    // Create a URL that navigates to the audio at the marker timestamp
    const markerUrl = `/audios/${props.marker.audio.id}?t=${props.marker.seconds}`;

    return (
      <GridCard
        className={`audio-marker-card ${zoomIndex()}`}
        url={markerUrl}
        title={markerTitle(props.marker)}
        width={props.cardWidth}
        linkClassName="audio-marker-card-link"
        thumbnailSectionClassName="audio-section"
        resumeTime={props.marker.seconds}
        image={<AudioMarkerCardImage {...props} />}
        details={<AudioMarkerCardDetails {...props} />}
        popovers={<AudioMarkerCardPopovers {...props} />}
        selected={props.selected}
        selecting={props.selecting}
        onSelectedChanged={props.onSelectedChanged}
      />
    );
  }
);
