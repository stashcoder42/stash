import React, { useEffect, useMemo, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useHistory, useLocation } from "react-router-dom";
import { AudioEditPanel } from "./AudioEditPanel";
import * as GQL from "src/core/generated-graphql";
import { mutateCreateAudio, useFindAudio } from "src/core/StashService";
import ImageUtils from "src/utils/image";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";

const AudioCreate: React.FC = () => {
  const history = useHistory();
  const intl = useIntl();
  const Toast = useToast();

  const location = useLocation();
  const query = useMemo(() => new URLSearchParams(location.search), [location]);

  // create audio from provided audio id if applicable
  const { data, loading } = useFindAudio(query.get("from_audio_id") ?? "new");
  const [loadingCoverImage, setLoadingCoverImage] = useState(false);
  const [coverImage, setCoverImage] = useState<string>();

  const audio = useMemo(() => {
    if (data?.findAudio) {
      return {
        ...data.findAudio,
        paths: undefined,
        id: undefined,
      };
    }

    return {
      title: query.get("q") ?? undefined,
    };
  }, [data?.findAudio, query]);

  useEffect(() => {
    async function fetchCoverImage() {
      const srcAudio = data?.findAudio;
      if (srcAudio?.paths.cover) {
        setLoadingCoverImage(true);
        const imageData = await ImageUtils.imageToDataURL(srcAudio.paths.cover);
        setCoverImage(imageData);
        setLoadingCoverImage(false);
      } else {
        setCoverImage(undefined);
      }
    }

    fetchCoverImage();
  }, [data?.findAudio]);

  if (loading || loadingCoverImage) {
    return <LoadingIndicator />;
  }

  async function onSave(input: GQL.AudioCreateInput) {
    const fileID = query.get("file_id") ?? undefined;
    const result = await mutateCreateAudio({
      ...input,
      file_ids: fileID ? [fileID] : undefined,
    });
    if (result.data?.audioCreate?.id) {
      history.push(`/audios/${result.data.audioCreate.id}`);
      Toast.success(
        intl.formatMessage(
          { id: "toast.created_entity" },
          { entity: intl.formatMessage({ id: "audio" }).toLocaleLowerCase() }
        )
      );
    }
  }

  return (
    <div className="row new-view justify-content-center" id="create-audio-page">
      <div className="col-md-8">
        <h2>
          <FormattedMessage
            id="actions.create_entity"
            values={{ entityType: intl.formatMessage({ id: "audio" }) }}
          />
        </h2>
        <AudioEditPanel
          audio={audio}
          initialCoverImage={coverImage}
          isVisible
          isNew
          onSubmit={onSave}
        />
      </div>
    </div>
  );
};

export default AudioCreate;
