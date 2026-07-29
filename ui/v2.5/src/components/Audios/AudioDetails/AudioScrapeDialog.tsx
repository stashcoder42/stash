import React, { useState } from "react";
import * as GQL from "src/core/generated-graphql";
import { ScrapeDialog } from "src/components/Shared/ScrapeDialog/ScrapeDialog";
import {
  ScrapedInputGroupRow,
  ScrapedTextAreaRow,
  ScrapedImageRow,
  ScrapedStringListRow,
} from "src/components/Shared/ScrapeDialog/ScrapeDialogRow";
import { useIntl } from "react-intl";
import { uniq } from "lodash-es";
import { Performer } from "src/components/Performers/PerformerSelect";
import { sortStoredIdObjects } from "src/utils/data";
import {
  ObjectListScrapeResult,
  ScrapeResult,
} from "src/components/Shared/ScrapeDialog/scrapeResult";
import { ScrapedPerformersRow } from "src/components/Shared/ScrapeDialog/ScrapedObjectsRow";
import { useCreateScrapedPerformer } from "src/components/Shared/ScrapeDialog/createObjects";
import { Tag } from "src/components/Tags/TagSelect";
import { useScrapedTags } from "src/components/Shared/ScrapeDialog/scrapedTags";

interface IAudioScrapeDialogProps {
  audio: Partial<GQL.AudioUpdateInput>;
  audioPerformers: Performer[];
  audioTags: Tag[];
  scraped: GQL.ScrapedAudio;

  onClose: (scrapedAudio?: GQL.ScrapedAudioDataFragment) => void;
}

export const AudioScrapeDialog: React.FC<IAudioScrapeDialogProps> = ({
  audio,
  audioPerformers,
  audioTags,
  scraped,
  onClose,
}) => {
  const [title, setTitle] = useState<ScrapeResult<string>>(
    new ScrapeResult<string>(audio.title, scraped.title)
  );

  const [urls, setURLs] = useState<ScrapeResult<string[]>>(
    new ScrapeResult<string[]>(
      audio.urls,
      scraped.urls
        ? uniq((audio.urls ?? []).concat(scraped.urls ?? []))
        : undefined
    )
  );

  const [date, setDate] = useState<ScrapeResult<string>>(
    new ScrapeResult<string>(audio.date, scraped.date)
  );

  const [performers, setPerformers] = useState<
    ObjectListScrapeResult<GQL.ScrapedPerformer>
  >(
    new ObjectListScrapeResult<GQL.ScrapedPerformer>(
      sortStoredIdObjects(
        audioPerformers.map((p) => ({
          stored_id: p.id,
          name: p.name,
        }))
      ),
      sortStoredIdObjects(scraped.performers ?? undefined)
    )
  );
  const [newPerformers, setNewPerformers] = useState<GQL.ScrapedPerformer[]>(
    scraped.performers?.filter((t) => !t.stored_id) ?? []
  );

  const { tags, newTags, scrapedTagsRow } = useScrapedTags(
    audioTags,
    scraped.tags
  );

  const [details, setDetails] = useState<ScrapeResult<string>>(
    new ScrapeResult<string>(audio.details, scraped.details)
  );

  const [image, setImage] = useState<ScrapeResult<string>>(
    new ScrapeResult<string>(audio.cover_image, scraped.image)
  );

  const createNewPerformer = useCreateScrapedPerformer({
    scrapeResult: performers,
    setScrapeResult: setPerformers,
    newObjects: newPerformers,
    setNewObjects: setNewPerformers,
  });

  const intl = useIntl();

  // don't show the dialog if nothing was scraped
  if (
    [title, urls, date, performers, tags, details, image].every(
      (r) => !r.scraped
    ) &&
    newTags.length === 0 &&
    newPerformers.length === 0
  ) {
    onClose();
    return <></>;
  }

  function makeNewScrapedItem(): GQL.ScrapedAudioDataFragment {
    return {
      title: title.getNewValue(),
      urls: urls.getNewValue(),
      date: date.getNewValue(),
      performers: performers.getNewValue(),
      tags: tags.getNewValue(),
      details: details.getNewValue(),
      image: image.getNewValue(),
    };
  }

  function renderScrapeRows() {
    return (
      <>
        <ScrapedInputGroupRow
          field="title"
          title={intl.formatMessage({ id: "title" })}
          result={title}
          onChange={(value) => setTitle(value)}
        />
        <ScrapedStringListRow
          field="urls"
          title={intl.formatMessage({ id: "urls" })}
          result={urls}
          onChange={(value) => setURLs(value)}
        />
        <ScrapedInputGroupRow
          field="date"
          title={intl.formatMessage({ id: "date" })}
          placeholder="YYYY-MM-DD"
          result={date}
          onChange={(value) => setDate(value)}
        />
        <ScrapedPerformersRow
          field="performers"
          title={intl.formatMessage({ id: "performers" })}
          result={performers}
          onChange={(value) => setPerformers(value)}
          newObjects={newPerformers}
          onCreateNew={createNewPerformer}
          ageFromDate={date.useNewValue ? date.newValue : date.originalValue}
        />
        {scrapedTagsRow}
        <ScrapedTextAreaRow
          field="details"
          title={intl.formatMessage({ id: "details" })}
          result={details}
          onChange={(value) => setDetails(value)}
        />
        <ScrapedImageRow
          field="cover_image"
          title={intl.formatMessage({ id: "cover_image" })}
          className="audio-cover"
          result={image}
          onChange={(value) => setImage(value)}
        />
      </>
    );
  }

  return (
    <ScrapeDialog
      title={intl.formatMessage(
        { id: "dialogs.scrape_entity_title" },
        { entity_type: intl.formatMessage({ id: "audio" }) }
      )}
      onClose={(apply) => {
        onClose(apply ? makeNewScrapedItem() : undefined);
      }}
    >
      {renderScrapeRows()}
    </ScrapeDialog>
  );
};

export default AudioScrapeDialog;
