import { Tab, Nav, Dropdown, Button } from "react-bootstrap";
import React, {
  useEffect,
  useState,
  useMemo,
  useRef,
  useLayoutEffect,
} from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useHistory, RouteComponentProps } from "react-router-dom";
import { Helmet } from "react-helmet";
import * as GQL from "src/core/generated-graphql";
import {
  mutateMetadataScan,
  useFindAudio,
  useAudioIncrementO,
  useAudioUpdate,
  queryFindAudios,
  queryFindAudiosByID,
  useAudioIncrementPlayCount,
  useAudioGenerateWaveform,
} from "src/core/StashService";

import { AudioEditPanel } from "./AudioEditPanel";
import { ErrorMessage } from "src/components/Shared/ErrorMessage";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { Icon } from "src/components/Shared/Icon";
import { Counter } from "src/components/Shared/Counter";
import { useToast } from "src/hooks/Toast";
import AudioQueue, { QueuedAudio } from "src/models/audioQueue";
import { ListFilterModel } from "src/models/list-filter/filter";
import Mousetrap from "mousetrap";
import { OrganizedButton } from "./OrganizedButton";
import { useConfigurationContext } from "src/hooks/Config";
import { FormattedDate } from "src/components/Shared/Date";
import { goBackOrReplace } from "src/utils/history";
import {
  faEllipsisV,
  faChevronRight,
  faChevronLeft,
} from "@fortawesome/free-solid-svg-icons";
import { objectPath, objectTitle } from "src/core/files";
import { RatingSystem } from "src/components/Shared/Rating/RatingSystem";
import TextUtils from "src/utils/text";
import {
  OCounterButton,
  ViewCountButton,
} from "src/components/Shared/CountButton";
import { useRatingKeybinds } from "src/hooks/keybinds";
import { lazyComponent } from "src/utils/lazyComponent";
import cx from "classnames";
import { TruncatedText } from "src/components/Shared/TruncatedText";
import { PatchComponent, PatchContainerComponent } from "src/patch";

const AudioPlayer = lazyComponent(
  () => import("src/components/AudioPlayer/AudioPlayer")
);

const ExternalPlayerButton = lazyComponent(
  () => import("./ExternalPlayerButton")
);

const QueueViewer = lazyComponent(() => import("./QueueViewer"));
const AudioMarkersPanel = lazyComponent(() => import("./AudioMarkersPanel"));
const AudioFileInfoPanel = lazyComponent(() => import("./AudioFileInfoPanel"));
const AudioDetailPanel = lazyComponent(() => import("./AudioDetailPanel"));
const AudioHistoryPanel = lazyComponent(() => import("./AudioHistoryPanel"));
const DeleteAudiosDialog = lazyComponent(() => import("../DeleteAudiosDialog"));

interface IProps {
  audio: GQL.AudioDataFragment;
  setTimestamp: (num: number) => void;
  queueAudios: QueuedAudio[];
  onQueueNext: () => void;
  onQueuePrevious: () => void;
  onQueueRandom: () => void;
  onQueueAudioClicked: (audioID: string) => void;
  onDelete: () => void;
  continuePlaylist: boolean;
  queueHasMoreAudios: boolean;
  onQueueMoreAudios: () => void;
  onQueueLessAudios: () => void;
  queueStart: number;
  collapsed: boolean;
  setCollapsed: (state: boolean) => void;
  setContinuePlaylist: (value: boolean) => void;
}

interface IAudioParams {
  id: string;
}

const AudioPageTabs = PatchContainerComponent<IProps>("AudioPage.Tabs");
const AudioPageTabContent = PatchContainerComponent<IProps>(
  "AudioPage.TabContent"
);

const AudioPage: React.FC<IProps> = PatchComponent("AudioPage", (props) => {
  const {
    audio,
    setTimestamp,
    queueAudios,
    onQueueNext,
    onQueuePrevious,
    onQueueRandom,
    onQueueAudioClicked,
    onDelete,
    continuePlaylist,
    queueHasMoreAudios,
    onQueueMoreAudios,
    onQueueLessAudios,
    queueStart,
    collapsed,
    setCollapsed,
    setContinuePlaylist,
  } = props;

  const Toast = useToast();
  const intl = useIntl();
  const [updateAudio] = useAudioUpdate();
  const { configuration } = useConfigurationContext();

  const [incrementO] = useAudioIncrementO(audio.id);

  const [incrementPlay] = useAudioIncrementPlayCount();

  function incrementPlayCount() {
    incrementPlay({
      variables: {
        id: audio.id,
      },
    });
  }

  const [organizedLoading, setOrganizedLoading] = useState(false);

  const [activeTabKey, setActiveTabKey] = useState("audio-details-panel");

  const [isDeleteAlertOpen, setIsDeleteAlertOpen] = useState<boolean>(false);

  const [generateWaveform] = useAudioGenerateWaveform();

  const onIncrementOClick = async () => {
    try {
      await incrementO();
    } catch (e) {
      Toast.error(e);
    }
  };

  function setRating(v: number | null) {
    updateAudio({
      variables: {
        input: {
          id: audio.id,
          rating100: v,
        },
      },
    });
  }

  useRatingKeybinds(
    true,
    configuration?.ui.ratingSystemOptions?.type,
    setRating
  );

  // set up hotkeys
  useEffect(() => {
    Mousetrap.bind("a", () => setActiveTabKey("audio-details-panel"));
    Mousetrap.bind("q", () => setActiveTabKey("audio-queue-panel"));
    Mousetrap.bind("e", () => setActiveTabKey("audio-edit-panel"));
    Mousetrap.bind("k", () => setActiveTabKey("audio-markers-panel"));
    Mousetrap.bind("i", () => setActiveTabKey("audio-file-info-panel"));
    Mousetrap.bind("h", () => setActiveTabKey("audio-history-panel"));
    Mousetrap.bind("o", () => {
      onIncrementOClick();
    });
    Mousetrap.bind("p n", () => onQueueNext());
    Mousetrap.bind("p p", () => onQueuePrevious());
    Mousetrap.bind("p r", () => onQueueRandom());
    Mousetrap.bind(",", () => setCollapsed(!collapsed));

    return () => {
      Mousetrap.unbind("a");
      Mousetrap.unbind("q");
      Mousetrap.unbind("e");
      Mousetrap.unbind("k");
      Mousetrap.unbind("i");
      Mousetrap.unbind("h");
      Mousetrap.unbind("o");
      Mousetrap.unbind("p n");
      Mousetrap.unbind("p p");
      Mousetrap.unbind("p r");
      Mousetrap.unbind(",");
    };
  });

  async function onSave(input: GQL.AudioCreateInput) {
    await updateAudio({
      variables: {
        input: {
          id: audio.id,
          ...input,
        },
      },
    });
    Toast.success(
      intl.formatMessage(
        { id: "toast.updated_entity" },
        { entity: intl.formatMessage({ id: "audio" }).toLocaleLowerCase() }
      )
    );
  }

  const onOrganizedClick = async () => {
    try {
      setOrganizedLoading(true);
      await updateAudio({
        variables: {
          input: {
            id: audio.id,
            organized: !audio.organized,
          },
        },
      });
    } catch (e) {
      Toast.error(e);
    } finally {
      setOrganizedLoading(false);
    }
  };

  function onClickMarker(marker: GQL.AudioMarkerDataFragment) {
    setTimestamp(marker.seconds);
  }

  async function onRescan() {
    await mutateMetadataScan({
      paths: [objectPath(audio)],
      rescan: true,
    });

    Toast.success(
      intl.formatMessage(
        { id: "toast.rescanning_entity" },
        {
          count: 1,
          singularEntity: intl
            .formatMessage({ id: "audio" })
            .toLocaleLowerCase(),
        }
      )
    );
  }

  async function onGenerateWaveform() {
    try {
      await generateWaveform({ variables: { id: audio.id } });
      Toast.success(intl.formatMessage({ id: "toast.generating_waveform" }));
    } catch (e) {
      Toast.error(e);
    }
  }

  function onDeleteDialogClosed(deleted: boolean) {
    setIsDeleteAlertOpen(false);
    if (deleted) {
      onDelete();
    }
  }

  function maybeRenderDeleteDialog() {
    if (isDeleteAlertOpen) {
      return (
        <DeleteAudiosDialog selected={[audio]} onClose={onDeleteDialogClosed} />
      );
    }
  }

  const renderOperations = () => (
    <Dropdown>
      <Dropdown.Toggle
        variant="secondary"
        id="operation-menu"
        className="minimal"
        title={intl.formatMessage({ id: "operations" })}
      >
        <Icon icon={faEllipsisV} />
      </Dropdown.Toggle>
      <Dropdown.Menu className="bg-secondary text-white">
        {!!audio.files.length && (
          <Dropdown.Item
            key="rescan"
            className="bg-secondary text-white"
            onClick={() => onRescan()}
          >
            <FormattedMessage id="actions.rescan" />
          </Dropdown.Item>
        )}
        <Dropdown.Item
          key="generate"
          className="bg-secondary text-white"
          onClick={() => onGenerateWaveform()}
        >
          <FormattedMessage id="actions.generate" />
        </Dropdown.Item>
        <Dropdown.Item
          key="delete-audio"
          className="bg-secondary text-white"
          onClick={() => setIsDeleteAlertOpen(true)}
        >
          <FormattedMessage
            id="actions.delete"
            values={{ entityType: intl.formatMessage({ id: "audio" }) }}
          />
        </Dropdown.Item>
      </Dropdown.Menu>
    </Dropdown>
  );

  const renderTabs = () => (
    <Tab.Container
      activeKey={activeTabKey}
      onSelect={(k) => k && setActiveTabKey(k)}
    >
      <div>
        <Nav variant="tabs" className="mr-auto">
          <AudioPageTabs {...props}>
            <Nav.Item>
              <Nav.Link eventKey="audio-details-panel">
                <FormattedMessage id="details" />
              </Nav.Link>
            </Nav.Item>
            {queueAudios.length > 0 ? (
              <Nav.Item>
                <Nav.Link eventKey="audio-queue-panel">
                  <FormattedMessage id="queue" />
                </Nav.Link>
              </Nav.Item>
            ) : (
              ""
            )}
            <Nav.Item>
              <Nav.Link eventKey="audio-markers-panel">
                <FormattedMessage id="markers" />
              </Nav.Link>
            </Nav.Item>
            {/* Audio doesn't have groups or galleries */}
            <Nav.Item>
              <Nav.Link eventKey="audio-file-info-panel">
                <FormattedMessage id="file_info" />
                <Counter count={audio.files.length} hideZero hideOne />
              </Nav.Link>
            </Nav.Item>
            <Nav.Item>
              <Nav.Link eventKey="audio-history-panel">
                <FormattedMessage id="history" />
              </Nav.Link>
            </Nav.Item>
            <Nav.Item>
              <Nav.Link eventKey="audio-edit-panel">
                <FormattedMessage id="actions.edit" />
              </Nav.Link>
            </Nav.Item>
          </AudioPageTabs>
        </Nav>
      </div>

      <Tab.Content>
        <AudioPageTabContent {...props}>
          <Tab.Pane eventKey="audio-details-panel">
            <AudioDetailPanel audio={audio} />
          </Tab.Pane>
          <Tab.Pane eventKey="audio-queue-panel">
            <QueueViewer
              audios={queueAudios}
              currentID={audio.id}
              continue={continuePlaylist}
              setContinue={setContinuePlaylist}
              onAudioClicked={onQueueAudioClicked}
              onNext={onQueueNext}
              onPrevious={onQueuePrevious}
              onRandom={onQueueRandom}
              start={queueStart}
              hasMoreAudios={queueHasMoreAudios}
              onLessAudios={onQueueLessAudios}
              onMoreAudios={onQueueMoreAudios}
            />
          </Tab.Pane>
          <Tab.Pane eventKey="audio-markers-panel">
            <AudioMarkersPanel
              audioId={audio.id}
              onClickMarker={onClickMarker}
              isVisible={activeTabKey === "audio-markers-panel"}
            />
          </Tab.Pane>
          <Tab.Pane
            className="file-info-panel"
            eventKey="audio-file-info-panel"
          >
            <AudioFileInfoPanel audio={audio} />
          </Tab.Pane>
          <Tab.Pane eventKey="audio-edit-panel" mountOnEnter>
            <AudioEditPanel
              isVisible={activeTabKey === "audio-edit-panel"}
              audio={audio}
              onSubmit={onSave}
              onDelete={() => setIsDeleteAlertOpen(true)}
            />
          </Tab.Pane>
          <Tab.Pane eventKey="audio-history-panel">
            <AudioHistoryPanel audio={audio} />
          </Tab.Pane>
        </AudioPageTabContent>
      </Tab.Content>
    </Tab.Container>
  );

  function getCollapseButtonIcon() {
    return collapsed ? faChevronRight : faChevronLeft;
  }

  const title = objectTitle(audio);

  const file = useMemo(
    () => (audio.files.length > 0 ? audio.files[0] : undefined),
    [audio]
  );

  return (
    <>
      <Helmet>
        <title>{title}</title>
      </Helmet>
      {maybeRenderDeleteDialog()}
      <div
        className={`audio-tabs order-xl-first order-last ${
          collapsed ? "collapsed" : ""
        }`}
      >
        <div>
          <div className="audio-header-container">
            {/* Audio doesn't have studios */}
            <h3 className="audio-header">
              <TruncatedText lineCount={2} text={title} />
            </h3>
          </div>

          <div className="audio-subheader">
            <span className="date" data-value={audio.date}>
              {!!audio.date && <FormattedDate value={audio.date} />}
            </span>
            {/* Audio doesn't need video resolution display */}
          </div>

          <div className="audio-toolbar">
            <span className="audio-toolbar-group">
              <RatingSystem
                value={audio.rating100}
                onSetRating={setRating}
                clickToRate
                withoutContext
              />
            </span>
            <span className="audio-toolbar-group">
              <span>
                <ExternalPlayerButton audio={audio} />
              </span>
              <span>
                <ViewCountButton
                  value={audio.play_count ?? 0}
                  onIncrement={() => incrementPlayCount()}
                />
              </span>
              <span>
                <OCounterButton
                  value={audio.o_counter ?? 0}
                  onIncrement={() => onIncrementOClick()}
                />
              </span>
              <span>
                <OrganizedButton
                  loading={organizedLoading}
                  organized={audio.organized}
                  onClick={onOrganizedClick}
                />
              </span>
              <span>{renderOperations()}</span>
            </span>
          </div>
        </div>
        {renderTabs()}
      </div>
      <div className="audio-divider d-none d-xl-block">
        <Button onClick={() => setCollapsed(!collapsed)}>
          <Icon className="fa-fw" icon={getCollapseButtonIcon()} />
        </Button>
      </div>
    </>
  );
});

const AudioLoader: React.FC<RouteComponentProps<IAudioParams>> = ({
  location,
  history,
  match,
}) => {
  const { id } = match.params;
  const { configuration } = useConfigurationContext();
  const { data, loading, error } = useFindAudio(id);

  const [audio, setAudio] = useState<GQL.AudioDataFragment>();

  // useLayoutEffect to update before paint
  useLayoutEffect(() => {
    // only update audio when loading is done
    if (!loading) {
      setAudio(data?.findAudio ?? undefined);
    }
  }, [data, loading]);

  const queryParams = useMemo(
    () => new URLSearchParams(location.search),
    [location.search]
  );
  const audioQueue = useMemo(
    () => AudioQueue.fromQueryParameters(queryParams),
    [queryParams]
  );
  const queryContinue = useMemo(() => {
    const cont = queryParams.get("continue");
    if (cont) {
      return cont === "true";
    } else {
      return !!configuration?.interface.continuePlaylistDefault;
    }
  }, [configuration?.interface.continuePlaylistDefault, queryParams]);

  const [queueAudios, setQueueAudios] = useState<QueuedAudio[]>([]);

  const [collapsed, setCollapsed] = useState(false);
  const [continuePlaylist, setContinuePlaylist] = useState(queryContinue);
  const [hideScrubber, setHideScrubber] = useState(
    !(configuration?.interface.showScrubber ?? true)
  );

  const _setTimestamp = useRef<(value: number) => void>();
  const initialTimestamp = useMemo(() => {
    const t = queryParams.get("t");
    if (!t) return 0;

    const n = Number(t);
    if (Number.isNaN(n)) return 0;
    return n;
  }, [queryParams]);

  const [queueTotal, setQueueTotal] = useState(0);
  const [queueStart, setQueueStart] = useState(1);

  const autoplay = queryParams.get("autoplay") === "true";
  const autoPlayOnSelected =
    configuration?.interface.autostartVideoOnPlaySelected ?? false;

  const currentQueueIndex = useMemo(
    () => queueAudios.findIndex((a) => a.id === id),
    [queueAudios, id]
  );

  function getSetTimestamp(fn: (value: number) => void) {
    _setTimestamp.current = fn;
  }

  function setTimestamp(value: number) {
    if (_setTimestamp.current) {
      _setTimestamp.current(value);
    }
  }

  // set up hotkeys
  useEffect(() => {
    Mousetrap.bind(".", () => setHideScrubber((value) => !value));

    return () => {
      Mousetrap.unbind(".");
    };
  }, []);

  async function getQueueFilterAudios(filter: ListFilterModel) {
    const query = await queryFindAudios(filter);
    const { audios, count } = query.data.findAudios;
    setQueueAudios(audios);
    setQueueTotal(count);
    setQueueStart((filter.currentPage - 1) * filter.itemsPerPage + 1);
  }

  async function getQueueAudios(audioIDs: number[]) {
    const query = await queryFindAudiosByID(audioIDs.map(String));
    const { audios, count } = query.data.findAudios;
    setQueueAudios(audios);
    setQueueTotal(count);
    setQueueStart(1);
  }

  useEffect(() => {
    if (audioQueue.query) {
      getQueueFilterAudios(audioQueue.query);
    } else if (audioQueue.audioIDs) {
      getQueueAudios(audioQueue.audioIDs);
    }
  }, [audioQueue]);

  async function onQueueLessAudios() {
    if (!audioQueue.query || queueStart <= 1) {
      return [];
    }

    const filterCopy = audioQueue.query.clone();
    const newStart = queueStart - filterCopy.itemsPerPage;
    filterCopy.currentPage = Math.ceil(newStart / filterCopy.itemsPerPage);
    const query = await queryFindAudios(filterCopy);
    const { audios } = query.data.findAudios;

    // prepend audios to audio list
    const newAudios = (audios as QueuedAudio[]).concat(queueAudios);
    setQueueAudios(newAudios);
    setQueueStart(newStart);

    return audios;
  }

  const queueHasMoreAudios = useMemo(() => {
    return queueStart + queueAudios.length - 1 < queueTotal;
  }, [queueStart, queueAudios, queueTotal]);

  async function onQueueMoreAudios() {
    if (!audioQueue.query || !queueHasMoreAudios) {
      return [];
    }

    const filterCopy = audioQueue.query.clone();
    const newStart = queueStart + queueAudios.length;
    filterCopy.currentPage = Math.ceil(newStart / filterCopy.itemsPerPage);
    const query = await queryFindAudios(filterCopy);
    const { audios } = query.data.findAudios;

    // append audios to audio list
    const newAudios = queueAudios.concat(audios ?? []);
    setQueueAudios(newAudios);
    // don't change queue start
    return audios ?? [];
  }

  function loadAudio(audioID: string, autoPlay?: boolean, newPage?: number) {
    const audioLink = audioQueue.makeLink(audioID, {
      newPage,
      autoPlay,
      continue: continuePlaylist,
    });
    history.replace(audioLink);
  }

  async function queueNext(autoPlay: boolean) {
    if (currentQueueIndex === -1) return;

    if (currentQueueIndex < queueAudios.length - 1) {
      loadAudio(queueAudios[currentQueueIndex + 1].id, autoPlay);
    } else {
      // if we're at the end of the queue, load more audios
      if (currentQueueIndex === queueAudios.length - 1 && queueHasMoreAudios) {
        const loadedAudios = await onQueueMoreAudios();
        if (loadedAudios && loadedAudios.length > 0) {
          // set the page to the next page
          const newPage = (audioQueue.query?.currentPage ?? 0) + 1;
          loadAudio(loadedAudios[0].id, autoPlay, newPage);
        }
      }
    }
  }

  async function queuePrevious(autoPlay: boolean) {
    if (currentQueueIndex === -1) return;

    if (currentQueueIndex > 0) {
      loadAudio(queueAudios[currentQueueIndex - 1].id, autoPlay);
    } else {
      // if we're at the beginning of the queue, load the previous page
      if (queueStart > 1) {
        const loadedAudios = await onQueueLessAudios();
        if (loadedAudios && loadedAudios.length > 0) {
          const newPage = (audioQueue.query?.currentPage ?? 0) - 1;
          loadAudio(
            loadedAudios[loadedAudios.length - 1].id,
            autoPlay,
            newPage
          );
        }
      }
    }
  }

  async function queueRandom(autoPlay: boolean) {
    if (audioQueue.query) {
      const { query } = audioQueue;
      const pages = Math.ceil(queueTotal / query.itemsPerPage);
      const page = Math.floor(Math.random() * pages) + 1;
      const index = Math.floor(
        Math.random() * Math.min(query.itemsPerPage, queueTotal)
      );
      const filterCopy = audioQueue.query.clone();
      filterCopy.currentPage = page;
      const queryResults = await queryFindAudios(filterCopy);
      if (queryResults.data.findAudios.audios.length > index) {
        const { id: audioID } = queryResults.data.findAudios.audios[index];
        // navigate to the audio player page
        loadAudio(audioID, autoPlay, page);
      }
    } else if (queueTotal !== 0) {
      const index = Math.floor(Math.random() * queueTotal);
      loadAudio(queueAudios[index].id, autoPlay);
    }
  }

  function onComplete() {
    // load the next scene if we're continuing
    if (continuePlaylist) {
      queueNext(true);
    }
  }

  function onDelete() {
    if (
      continuePlaylist &&
      currentQueueIndex >= 0 &&
      currentQueueIndex < queueAudios.length - 1
    ) {
      loadAudio(queueAudios[currentQueueIndex + 1].id);
    } else {
      goBackOrReplace(history, "/audios");
    }
  }

  function getAudioPage(audioID: string) {
    if (!audioQueue.query) return;

    // find the page that the audio is on
    const index = queueAudios.findIndex((a) => a.id === audioID);

    if (index === -1) return;

    const perPage = audioQueue.query.itemsPerPage;
    return Math.floor((index + queueStart - 1) / perPage) + 1;
  }

  function onQueueAudioClicked(audioID: string) {
    loadAudio(audioID, autoPlayOnSelected, getAudioPage(audioID));
  }

  if (!audio) {
    if (loading) return <LoadingIndicator />;
    if (error) return <ErrorMessage error={error.message} />;
    return <ErrorMessage error={`No audio found with id ${id}.`} />;
  }

  return (
    <div className="row">
      <AudioPage
        audio={audio}
        setTimestamp={setTimestamp}
        queueAudios={queueAudios}
        queueStart={queueStart}
        onDelete={onDelete}
        onQueueNext={() => queueNext(autoPlayOnSelected)}
        onQueuePrevious={() => queuePrevious(autoPlayOnSelected)}
        onQueueRandom={() => queueRandom(autoPlayOnSelected)}
        onQueueAudioClicked={onQueueAudioClicked}
        continuePlaylist={continuePlaylist}
        queueHasMoreAudios={queueHasMoreAudios}
        onQueueLessAudios={onQueueLessAudios}
        onQueueMoreAudios={onQueueMoreAudios}
        collapsed={collapsed}
        setCollapsed={setCollapsed}
        setContinuePlaylist={setContinuePlaylist}
      />
      <div className={`audio-player-container ${collapsed ? "expanded" : ""}`}>
        <AudioPlayer
          key="AudioPlayer"
          audio={audio}
          hideScrubberOverride={hideScrubber}
          autoplay={autoplay}
          permitLoop={!continuePlaylist}
          initialTimestamp={initialTimestamp}
          sendSetTimestamp={getSetTimestamp}
          onComplete={onComplete}
          onNext={() => queueNext(true)}
          onPrevious={() => queuePrevious(true)}
        />
      </div>
    </div>
  );
};

export default AudioLoader;
