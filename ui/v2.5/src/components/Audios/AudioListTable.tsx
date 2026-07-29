import React from "react";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { AudioQueue } from "src/models/audioQueue";
import { IColumn, ListTable } from "../List/ListTable";
import { TagLink } from "../Shared/TagLink";
import { TruncatedText } from "../Shared/TruncatedText";
import { PerformerPopoverButton } from "../Shared/PerformerPopoverButton";
import TextUtils from "src/utils/text";
import { useIntl } from "react-intl";
import { useTableColumns } from "src/hooks/useTableColumns";
import { RatingSystem } from "../Shared/Rating/RatingSystem";
import { useAudioUpdate } from "src/core/StashService";

interface IAudioListTable {
  audios: GQL.SlimAudioDataFragment[];
  queue: AudioQueue;
  selectedIds: Set<string>;
  onSelectChange: (id: string, selected: boolean, shiftKey: boolean) => void;
}

const TABLE_NAME = "audios";

export const AudioListTable: React.FC<IAudioListTable> = ({
  audios,
  queue,
  selectedIds,
  onSelectChange,
}) => {
  const intl = useIntl();
  const [updateAudio] = useAudioUpdate();

  function setRating(v: number | null, audioId: string) {
    if (audioId) {
      updateAudio({
        variables: {
          input: {
            id: audioId,
            rating100: v,
          },
        },
      });
    }
  }

  // Cell render functions
  const CoverImageCell = (audio: GQL.SlimAudioDataFragment, index: number) => {
    return (
      <Link to={queue.makeLink(audio.id, { audioIndex: index })}>
        <img
          className="audio-table-thumb"
          src={audio.paths.cover || ""}
          alt={audio.title || ""}
          style={{ width: "40px", height: "40px", objectFit: "cover" }}
        />
      </Link>
    );
  };

  const TitleCell = (audio: GQL.SlimAudioDataFragment, index: number) => {
    return (
      <Link to={queue.makeLink(audio.id, { audioIndex: index })}>
        <TruncatedText text={audio.title || ""} />
      </Link>
    );
  };

  const PerformersCell = (audio: GQL.SlimAudioDataFragment) => {
    if (audio.performers.length === 0) {
      return null;
    }

    // PerformerPopoverButton expects an array of performers
    return (
      <div className="performers-list">
        <PerformerPopoverButton performers={audio.performers} />
      </div>
    );
  };

  const TagsCell = (audio: GQL.SlimAudioDataFragment) => {
    return (
      <div className="tags-list">
        {audio.tags.slice(0, 3).map((tag) => (
          <TagLink key={tag.id} tag={tag} linkType="audio" />
        ))}
        {audio.tags.length > 3 && (
          <span className="text-muted">+{audio.tags.length - 3} more</span>
        )}
      </div>
    );
  };

  const DurationCell = (audio: GQL.SlimAudioDataFragment) => {
    const file = audio.files.length > 0 ? audio.files[0] : undefined;
    const duration = file?.duration
      ? TextUtils.secondsAsTimeString(file.duration)
      : "";
    return <>{duration}</>;
  };

  const DateCell = (audio: GQL.SlimAudioDataFragment) => {
    return <>{audio.date}</>;
  };

  const RatingCell = (audio: GQL.SlimAudioDataFragment) => {
    return (
      <RatingSystem
        value={audio.rating100}
        onSetRating={(value) => setRating(value, audio.id)}
        clickToRate
      />
    );
  };

  const OrganizedCell = (audio: GQL.SlimAudioDataFragment) => {
    return (
      <span className="text-center">
        {audio.organized && <span className="text-success">✓</span>}
      </span>
    );
  };

  // Column definitions
  interface IColumnSpec {
    value: string;
    label: string;
    defaultShow?: boolean;
    mandatory?: boolean;
    render?: (
      audio: GQL.SlimAudioDataFragment,
      index: number
    ) => React.ReactNode;
  }

  const allColumns: IColumnSpec[] = [
    {
      value: "cover",
      label: "",
      defaultShow: true,
      render: CoverImageCell,
    },
    {
      value: "title",
      label: intl.formatMessage({ id: "title" }),
      defaultShow: true,
      mandatory: true,
      render: TitleCell,
    },
    {
      value: "performers",
      label: intl.formatMessage({ id: "performers" }),
      defaultShow: true,
      render: PerformersCell,
    },
    {
      value: "tags",
      label: intl.formatMessage({ id: "tags" }),
      defaultShow: true,
      render: TagsCell,
    },
    {
      value: "duration",
      label: intl.formatMessage({ id: "duration" }),
      defaultShow: true,
      render: DurationCell,
    },
    {
      value: "date",
      label: intl.formatMessage({ id: "date" }),
      defaultShow: true,
      render: DateCell,
    },
    {
      value: "rating",
      label: intl.formatMessage({ id: "rating" }),
      defaultShow: true,
      render: RatingCell,
    },
    {
      value: "organized",
      label: "O",
      defaultShow: true,
      render: OrganizedCell,
    },
  ];

  const defaultColumns = allColumns
    .filter((col) => col.defaultShow)
    .map((col) => col.value);

  const { selectedColumns, saveColumns } = useTableColumns(
    TABLE_NAME,
    defaultColumns
  );

  // Create render function map
  const columnRenderFuncs: Record<
    string,
    (audio: GQL.SlimAudioDataFragment, index: number) => React.ReactNode
  > = {};
  allColumns.forEach((col) => {
    if (col.render) {
      columnRenderFuncs[col.value] = col.render;
    }
  });

  function renderCell(
    column: IColumn,
    audio: GQL.SlimAudioDataFragment,
    index: number
  ) {
    const render = columnRenderFuncs[column.value];
    if (render) return render(audio, index);
  }

  return (
    <ListTable
      className="audio-table"
      items={audios}
      allColumns={allColumns}
      columns={selectedColumns}
      setColumns={(c) => saveColumns(c)}
      selectedIds={selectedIds}
      onSelectChange={onSelectChange}
      renderCell={renderCell}
    />
  );
};
