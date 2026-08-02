import React, { useEffect, useState, useMemo } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { Button, Form, Col, Row } from "react-bootstrap";
import Mousetrap from "mousetrap";
import * as GQL from "src/core/generated-graphql";
import * as yup from "yup";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { ImageInput } from "src/components/Shared/ImageInput";
import { useToast } from "src/hooks/Toast";
import ImageUtils from "src/utils/image";
import { useFormik } from "formik";
import { Prompt } from "react-router-dom";
import isEqual from "lodash-es/isEqual";
import {
  yupDateString,
  yupFormikValidate,
  yupUniqueStringList,
} from "src/utils/yup";
import {
  Performer,
  PerformerSelect,
} from "src/components/Performers/PerformerSelect";
import { formikUtils } from "src/utils/form";
import { useTagsEdit } from "src/hooks/tagsEdit";
import {
  useListAudioScrapers,
  queryScrapeAudioURL,
} from "src/core/StashService";
import { lazyComponent } from "src/utils/lazyComponent";

const AudioScrapeDialog = lazyComponent(() => import("./AudioScrapeDialog"));

interface IProps {
  audio: Partial<GQL.AudioDataFragment>;
  initialCoverImage?: string;
  isNew?: boolean;
  isVisible: boolean;
  onSubmit: (input: GQL.AudioCreateInput) => Promise<void>;
  onDelete?: () => void;
}

export const AudioEditPanel: React.FC<IProps> = ({
  audio,
  initialCoverImage,
  isNew = false,
  isVisible,
  onSubmit,
  onDelete,
}) => {
  const intl = useIntl();
  const Toast = useToast();

  const [performers, setPerformers] = useState<Performer[]>([]);
  const [scrapedAudio, setScrapedAudio] = useState<GQL.ScrapedAudio | null>();

  const Scrapers = useListAudioScrapers();

  useEffect(() => {
    setPerformers(audio.performers ?? []);
  }, [audio.performers]);

  // Network state
  const [isLoading, setIsLoading] = useState(false);

  const schema = yup.object({
    title: yup.string().ensure(),
    urls: yupUniqueStringList(intl),
    date: yupDateString(intl),
    performer_ids: yup.array(yup.string().required()).defined(),
    tag_ids: yup.array(yup.string().required()).defined(),
    details: yup.string().ensure(),
    cover_image: yup.string().nullable().optional(),
  });

  const initialValues = useMemo(
    () => ({
      title: audio.title ?? "",
      urls: audio.urls ?? [],
      date: audio.date ?? "",
      performer_ids: (audio.performers ?? []).map((p) => p.id),
      tag_ids: (audio.tags ?? []).map((t) => t.id),
      details: audio.details ?? "",
      cover_image: initialCoverImage,
    }),
    [audio, initialCoverImage]
  );

  type InputValues = yup.InferType<typeof schema>;

  const formik = useFormik<InputValues>({
    initialValues,
    enableReinitialize: true,
    validate: yupFormikValidate(schema),
    onSubmit: (values) => onSave(schema.cast(values)),
  });

  const { tags, updateTagsStateFromScraper, tagsControl } = useTagsEdit(
    audio.tags,
    (ids) => formik.setFieldValue("tag_ids", ids)
  );

  const coverImagePreview = useMemo(() => {
    const audioImage = audio.paths?.cover;
    const formImage = formik.values.cover_image;
    if (formImage === null && audioImage) {
      const audioImageURL = new URL(audioImage);
      audioImageURL.searchParams.set("default", "true");
      return audioImageURL.toString();
    } else if (formImage) {
      return formImage;
    }
    return audioImage;
  }, [formik.values.cover_image, audio.paths?.cover]);

  function onSetPerformers(items: Performer[]) {
    setPerformers(items);
    formik.setFieldValue(
      "performer_ids",
      items.map((item) => item.id)
    );
  }

  useEffect(() => {
    if (isVisible) {
      Mousetrap.bind("s s", () => {
        if (formik.dirty) {
          formik.submitForm();
        }
      });
      Mousetrap.bind("d d", () => {
        if (onDelete) {
          onDelete();
        }
      });

      return () => {
        Mousetrap.unbind("s s");
        Mousetrap.unbind("d d");
      };
    }
  });

  async function onSave(input: InputValues) {
    setIsLoading(true);
    try {
      await onSubmit(input);
      formik.resetForm();
    } catch (e) {
      Toast.error(e);
    }
    setIsLoading(false);
  }

  const encodingImage = ImageUtils.usePasteImage(onImageLoad);

  function onImageLoad(imageData: string) {
    formik.setFieldValue("cover_image", imageData);
  }

  function onCoverImageChange(event: React.FormEvent<HTMLInputElement>) {
    ImageUtils.onImageChange(event, onImageLoad);
  }

  const image = useMemo(() => {
    if (encodingImage) {
      return (
        <LoadingIndicator
          message={intl.formatMessage({ id: "actions.encoding_image" })}
        />
      );
    }

    if (coverImagePreview) {
      return (
        <img
          className="audio-cover"
          src={coverImagePreview}
          alt={intl.formatMessage({ id: "cover_image" })}
        />
      );
    }

    return <div></div>;
  }, [encodingImage, coverImagePreview, intl]);

  if (isLoading) return <LoadingIndicator />;

  const splitProps = {
    labelProps: {
      column: true,
      sm: 3,
    },
    fieldProps: {
      sm: 9,
    },
  };
  const fullWidthProps = {
    labelProps: {
      column: true,
      sm: 3,
      xl: 12,
    },
    fieldProps: {
      sm: 9,
      xl: 12,
    },
  };
  const { renderField, renderInputField, renderDateField, renderURLListField } =
    formikUtils(intl, formik, splitProps);

  function renderPerformersField() {
    const date = (() => {
      try {
        return schema.validateSyncAt("date", formik.values);
      } catch {
        return undefined;
      }
    })();

    const title = intl.formatMessage({ id: "performers" });
    const control = (
      <PerformerSelect
        isMulti
        onSelect={onSetPerformers}
        values={performers}
        ageFromDate={date}
      />
    );

    return renderField("performer_ids", title, control, fullWidthProps);
  }

  function renderTagsField() {
    const title = intl.formatMessage({ id: "tags" });
    return renderField("tag_ids", title, tagsControl(), fullWidthProps);
  }

  function renderDetailsField() {
    const props = {
      labelProps: {
        column: true,
        sm: 3,
        lg: 12,
      },
      fieldProps: {
        sm: 9,
        lg: 12,
      },
    };

    return renderInputField("details", "textarea", "details", props);
  }

  async function onScrapeAudioURL(url: string) {
    if (!url) {
      return;
    }
    setIsLoading(true);
    try {
      const result = await queryScrapeAudioURL(url);
      if (!result.data?.scrapeAudioURL) {
        return;
      }
      setScrapedAudio(result.data.scrapeAudioURL);
    } catch (e) {
      Toast.error(e);
    } finally {
      setIsLoading(false);
    }
  }

  function urlScrapable(scrapedUrl: string): boolean {
    return (Scrapers?.data?.listScrapers ?? []).some((s) =>
      (s?.audio?.urls ?? []).some((u) => scrapedUrl.includes(u))
    );
  }

  function onScrapeDialogClosed(audioData?: GQL.ScrapedAudioDataFragment) {
    if (audioData) {
      updateAudioFromScrapedAudio(audioData);
    }
    setScrapedAudio(undefined);
  }

  function updateAudioFromScrapedAudio(
    updatedAudio: GQL.ScrapedAudioDataFragment
  ) {
    if (updatedAudio.title) {
      formik.setFieldValue("title", updatedAudio.title);
    }

    if (updatedAudio.details) {
      formik.setFieldValue("details", updatedAudio.details);
    }

    if (updatedAudio.date) {
      formik.setFieldValue("date", updatedAudio.date);
    }

    if (updatedAudio.urls) {
      formik.setFieldValue("urls", updatedAudio.urls);
    }

    // update performers - only those with stored_id
    if (updatedAudio.performers?.length) {
      const idPerfs = updatedAudio.performers.filter((p) => {
        return p.stored_id !== undefined && p.stored_id !== null;
      });

      if (idPerfs.length > 0) {
        onSetPerformers(
          idPerfs.map((p) => {
            return {
              id: p.stored_id!,
              name: p.name ?? "",
              alias_list: [],
              favorite: false,
              disambiguation: p.disambiguation ?? "",
            };
          })
        );
      }
    }

    updateTagsStateFromScraper(updatedAudio.tags ?? undefined);

    if (updatedAudio.image) {
      // image is a base64 string
      formik.setFieldValue("cover_image", updatedAudio.image);
    }
  }

  function maybeRenderScrapeDialog() {
    if (!scrapedAudio) {
      return;
    }

    const currentAudio = {
      id: audio.id,
      ...formik.values,
    };

    if (!currentAudio.cover_image) {
      currentAudio.cover_image = audio.paths?.cover;
    }

    return (
      <AudioScrapeDialog
        audio={currentAudio}
        audioTags={tags}
        audioPerformers={performers}
        scraped={scrapedAudio}
        onClose={(s) => onScrapeDialogClosed(s)}
      />
    );
  }

  return (
    <div id="audio-edit-details">
      {maybeRenderScrapeDialog()}
      <Prompt
        when={formik.dirty}
        message={intl.formatMessage({ id: "dialogs.unsaved_changes" })}
      />

      <Form noValidate onSubmit={formik.handleSubmit}>
        <Row className="form-container edit-buttons-container px-3 pt-3">
          <div className="edit-buttons mb-3 pl-0">
            <Button
              className="edit-button"
              variant="primary"
              disabled={
                (!isNew && !formik.dirty) || !isEqual(formik.errors, {})
              }
              onClick={() => formik.submitForm()}
            >
              <FormattedMessage id="actions.save" />
            </Button>
            {onDelete && (
              <Button
                className="edit-button"
                variant="danger"
                onClick={() => onDelete()}
              >
                <FormattedMessage id="actions.delete" />
              </Button>
            )}
          </div>
        </Row>
        <Row className="form-container px-3">
          <Col lg={7} xl={12}>
            {renderInputField("title")}

            {renderURLListField("urls", onScrapeAudioURL, urlScrapable)}

            {renderDateField("date")}

            {renderPerformersField()}
            {renderTagsField()}
          </Col>
          <Col lg={5} xl={12}>
            {renderDetailsField()}
            <Form.Group controlId="cover_image">
              <Form.Label>
                <FormattedMessage id="cover_image" />
              </Form.Label>
              {image}
              <ImageInput
                isEditing
                onImageChange={onCoverImageChange}
                onImageURL={onImageLoad}
              />
            </Form.Group>
          </Col>
        </Row>
      </Form>
    </div>
  );
};

export default AudioEditPanel;
