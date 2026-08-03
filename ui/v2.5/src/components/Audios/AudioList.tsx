import React, { useCallback, useEffect, useMemo } from "react";
import cloneDeep from "lodash-es/cloneDeep";
import { FormattedMessage, useIntl } from "react-intl";
import { useHistory } from "react-router-dom";
import Mousetrap from "mousetrap";
import * as GQL from "src/core/generated-graphql";
import { queryFindAudios, useFindAudios } from "src/core/StashService";
import { ListFilterModel } from "src/models/list-filter/filter";
import { DisplayMode } from "src/models/list-filter/types";
import { IPlayAudioOptions, AudioQueue } from "src/models/audioQueue";
import { AudioListTable } from "./AudioListTable";
import { EditAudiosDialog } from "./EditAudiosDialog";
import { DeleteAudiosDialog } from "./DeleteAudiosDialog";
import { ExportDialog } from "../Shared/ExportDialog";
import { AudioCardsGrid } from "./AudioCardsGrid";
import { AudioWallPanel } from "./AudioWallPanel";
import { useConfigurationContext } from "src/hooks/Config";
import {
  faPencil,
  faPlay,
  faPlus,
  faTrash,
} from "@fortawesome/free-solid-svg-icons";
import TextUtils from "src/utils/text";
import { View } from "../List/views";
import { FileSize } from "../Shared/FileSize";
import { LoadedContent } from "../List/PagedList";
import { useCloseEditDelete, useFilterOperations } from "../List/util";
import {
  OperationDropdown,
  OperationDropdownItem,
} from "../List/ListOperationButtons";
import { useFilteredItemList } from "../List/ItemList";
import {
  Sidebar,
  SidebarPane,
  SidebarPaneContent,
  SidebarStateContext,
  useSidebarState,
} from "../Shared/Sidebar";
import { SidebarPerformersFilter } from "../List/Filters/PerformersFilter";
import { PerformersCriterionOption } from "src/models/list-filter/criteria/performers";
import { TagsCriterionOption } from "src/models/list-filter/criteria/tags";
import { SidebarTagsFilter } from "../List/Filters/TagsFilter";
import cx from "classnames";
import { RatingCriterionOption } from "src/models/list-filter/criteria/rating";
import { SidebarRatingFilter } from "../List/Filters/RatingFilter";
import { OrganizedCriterionOption } from "src/models/list-filter/criteria/organized";
import { HasMarkersCriterionOption } from "src/models/list-filter/criteria/has-markers";
import { SidebarBooleanFilter } from "../List/Filters/BooleanFilter";
import { DurationCriterionOption } from "src/models/list-filter/audios";
import { SidebarDurationFilter } from "../List/Filters/SidebarDurationFilter";
import {
  FilteredSidebarHeader,
  useFilteredSidebarKeybinds,
} from "../List/Filters/FilterSidebar";
import { PatchComponent, PatchContainerComponent } from "src/patch";
import { Pagination, PaginationIndex } from "../List/Pagination";
import { Button, ButtonGroup } from "react-bootstrap";
import { Icon } from "../Shared/Icon";
import useFocus from "src/utils/focus";
import { useZoomKeybinds } from "../List/ZoomSlider";
import { FilteredListToolbar } from "../List/FilteredListToolbar";
import { FilterTags } from "../List/FilterTags";

function renderMetadataByline(result: GQL.FindAudiosQueryResult) {
  const duration = result?.data?.findAudios?.duration;
  const size = result?.data?.findAudios?.filesize;

  if (!duration && !size) {
    return;
  }

  const separator = duration && size ? " - " : "";

  return (
    <span className="audios-stats">
      &nbsp;(
      {duration ? (
        <span className="audios-duration">
          {TextUtils.secondsAsTimeString(duration, 3)}
        </span>
      ) : undefined}
      {separator}
      {size ? (
        <span className="audios-size">
          <FileSize size={size} />
        </span>
      ) : undefined}
      )
    </span>
  );
}

function usePlayAudio() {
  const history = useHistory();

  const { configuration: config } = useConfigurationContext();
  const cont = config?.interface.continuePlaylistDefault ?? false;
  const autoPlay = config?.interface.autostartVideoOnPlaySelected ?? false;

  const playAudio = useCallback(
    (queue: AudioQueue, audioID: string, options?: IPlayAudioOptions) => {
      history.push(
        queue.makeLink(audioID, { autoPlay, continue: cont, ...options })
      );
    },
    [history, cont, autoPlay]
  );

  return playAudio;
}

function usePlaySelected(selectedIds: Set<string>) {
  const playAudio = usePlayAudio();

  const playSelected = useCallback(() => {
    // populate queue and go to first audio
    const audioIDs = Array.from(selectedIds.values());
    const queue = AudioQueue.fromAudioIDList(audioIDs);

    playAudio(queue, audioIDs[0]);
  }, [selectedIds, playAudio]);

  return playSelected;
}

function usePlayFirst() {
  const playAudio = usePlayAudio();

  const playFirst = useCallback(
    (queue: AudioQueue, audioID: string, index: number) => {
      // populate queue and go to first audio
      playAudio(queue, audioID, { audioIndex: index });
    },
    [playAudio]
  );

  return playFirst;
}

function usePlayRandom(filter: ListFilterModel, count: number) {
  const playAudio = usePlayAudio();

  const playRandom = useCallback(async () => {
    // query for a random audio
    if (count === 0) {
      return;
    }

    const pages = Math.ceil(count / filter.itemsPerPage);
    const page = Math.floor(Math.random() * pages) + 1;

    const indexMax = Math.min(filter.itemsPerPage, count);
    const index = Math.floor(Math.random() * indexMax);
    const filterCopy = cloneDeep(filter);
    filterCopy.currentPage = page;
    filterCopy.sortBy = "random";
    const queryResults = await queryFindAudios(filterCopy);
    const audio = queryResults.data.findAudios.audios[index];
    if (audio) {
      // navigate to the audio player page
      const queue = AudioQueue.fromListFilterModel(filterCopy);
      playAudio(queue, audio.id, { audioIndex: index });
    }
  }, [filter, count, playAudio]);

  return playRandom;
}

function useAddKeybinds(filter: ListFilterModel, count: number) {
  const playRandom = usePlayRandom(filter, count);

  useEffect(() => {
    Mousetrap.bind("p r", () => {
      playRandom();
    });

    return () => {
      Mousetrap.unbind("p r");
    };
  }, [playRandom]);
}

const AudioList: React.FC<{
  audios: GQL.SlimAudioDataFragment[];
  filter: ListFilterModel;
  selectedIds: Set<string>;
  onSelectChange: (id: string, selected: boolean, _shiftKey: boolean) => void;
  fromGroupId?: string;
}> = ({ audios, filter, selectedIds, onSelectChange }) => {
  const queue = useMemo(() => AudioQueue.fromListFilterModel(filter), [filter]);

  if (audios.length === 0) {
    return null;
  }

  if (filter.displayMode === DisplayMode.Grid) {
    return (
      <AudioCardsGrid
        audios={audios}
        queue={queue}
        zoomIndex={filter.zoomIndex}
        selectedIds={selectedIds}
        onSelectChange={onSelectChange}
      />
    );
  }
  if (filter.displayMode === DisplayMode.List) {
    return (
      <AudioListTable
        audios={audios}
        queue={queue}
        selectedIds={selectedIds}
        onSelectChange={onSelectChange}
      />
    );
  }
  if (filter.displayMode === DisplayMode.Wall) {
    return (
      <AudioWallPanel
        audios={audios}
        audioQueue={queue}
        zoomIndex={filter.zoomIndex}
        selectedIds={selectedIds}
        onSelectChange={onSelectChange}
      />
    );
  }

  return null;
};

const AudiosFilterSidebarSections = PatchContainerComponent(
  "FilteredAudioList.SidebarSections"
);

const SidebarContent: React.FC<{
  filter: ListFilterModel;
  setFilter: (filter: ListFilterModel) => void;
  filterHook?: (filter: ListFilterModel) => ListFilterModel;
  view?: View;
  sidebarOpen: boolean;
  onClose?: () => void;
  showEditFilter: (editingCriterion?: string) => void;
  count?: number;
  focus?: ReturnType<typeof useFocus>;
}> = ({
  filter,
  setFilter,
  filterHook,
  view,
  showEditFilter,
  sidebarOpen,
  onClose,
  count,
  focus,
}) => {
  const showResultsId =
    count !== undefined ? "actions.show_count_results" : "actions.show_results";

  return (
    <>
      <FilteredSidebarHeader
        sidebarOpen={sidebarOpen}
        showEditFilter={showEditFilter}
        filter={filter}
        setFilter={setFilter}
        view={view}
        focus={focus}
      />

      <AudiosFilterSidebarSections>
        <SidebarPerformersFilter
          title={<FormattedMessage id="performers" />}
          data-type={PerformersCriterionOption.type}
          option={PerformersCriterionOption}
          filter={filter}
          setFilter={setFilter}
          filterHook={filterHook}
          sectionID="performers"
        />
        <SidebarTagsFilter
          title={<FormattedMessage id="tags" />}
          data-type={TagsCriterionOption.type}
          option={TagsCriterionOption}
          filter={filter}
          setFilter={setFilter}
          filterHook={filterHook}
          sectionID="tags"
        />
        <SidebarRatingFilter
          title={<FormattedMessage id="rating" />}
          data-type={RatingCriterionOption.type}
          option={RatingCriterionOption}
          filter={filter}
          setFilter={setFilter}
          sectionID="rating"
        />
        <SidebarDurationFilter
          title={<FormattedMessage id="duration" />}
          option={DurationCriterionOption}
          filter={filter}
          setFilter={setFilter}
          sectionID="duration"
        />
        <SidebarBooleanFilter
          title={<FormattedMessage id="hasMarkers" />}
          data-type={HasMarkersCriterionOption.type}
          option={HasMarkersCriterionOption}
          filter={filter}
          setFilter={setFilter}
          sectionID="hasMarkers"
        />
        <SidebarBooleanFilter
          title={<FormattedMessage id="organized" />}
          data-type={OrganizedCriterionOption.type}
          option={OrganizedCriterionOption}
          filter={filter}
          setFilter={setFilter}
          sectionID="organized"
        />
      </AudiosFilterSidebarSections>

      <div className="sidebar-footer">
        <Button className="sidebar-close-button" onClick={onClose}>
          <FormattedMessage id={showResultsId} values={{ count }} />
        </Button>
      </div>
    </>
  );
};

interface IOperations {
  text: string;
  onClick: () => void;
  isDisplayed?: () => boolean;
  className?: string;
}

const AudioListOperations: React.FC<{
  items: number;
  hasSelection: boolean;
  operations: IOperations[];
  onEdit: () => void;
  onDelete: () => void;
  onPlay: () => void;
  onCreateNew: () => void;
}> = PatchComponent(
  "AudioListOperations",
  ({
    items,
    hasSelection,
    operations,
    onEdit,
    onDelete,
    onPlay,
    onCreateNew,
  }) => {
    const intl = useIntl();

    return (
      <div className="audio-list-operations">
        <ButtonGroup>
          {!!items && (
            <Button
              className="play-button"
              variant="secondary"
              onClick={() => onPlay()}
              title={intl.formatMessage({ id: "actions.play" })}
            >
              <Icon icon={faPlay} />
            </Button>
          )}
          {!hasSelection && (
            <Button
              className="create-new-button"
              variant="secondary"
              onClick={() => onCreateNew()}
              title={intl.formatMessage(
                { id: "actions.create_entity" },
                { entityType: intl.formatMessage({ id: "audio" }) }
              )}
            >
              <Icon icon={faPlus} />
            </Button>
          )}

          {hasSelection && (
            <>
              <Button variant="secondary" onClick={() => onEdit()}>
                <Icon icon={faPencil} />
              </Button>
              <Button
                variant="danger"
                className="btn-danger-minimal"
                onClick={() => onDelete()}
              >
                <Icon icon={faTrash} />
              </Button>
            </>
          )}

          <OperationDropdown
            className="audio-list-operations"
            menuClassName="audio-list-operations-dropdown"
            menuPortalTarget={document.body}
          >
            {operations.map((o) => {
              if (o.isDisplayed && !o.isDisplayed()) {
                return null;
              }

              return (
                <OperationDropdownItem
                  key={o.text}
                  onClick={o.onClick}
                  text={o.text}
                  className={o.className}
                />
              );
            })}
          </OperationDropdown>
        </ButtonGroup>
      </div>
    );
  }
);

interface IFilteredAudios {
  filterHook?: (filter: ListFilterModel) => ListFilterModel;
  defaultSort?: string;
  view?: View;
  alterQuery?: boolean;
  fromGroupId?: string;
}

export const FilteredAudioList = (props: IFilteredAudios) => {
  const intl = useIntl();
  const history = useHistory();

  const searchFocus = useFocus();

  const { filterHook, defaultSort, view, alterQuery, fromGroupId } = props;

  // States
  const {
    showSidebar,
    setShowSidebar,
    loading: sidebarStateLoading,
    sectionOpen,
    setSectionOpen,
  } = useSidebarState(view);

  const { filterState, queryResult, modalState, listSelect, showEditFilter } =
    useFilteredItemList({
      filterStateProps: {
        filterMode: GQL.FilterMode.Audios,
        defaultSort,
        view,
        useURL: alterQuery,
      },
      queryResultProps: {
        useResult: useFindAudios,
        getCount: (r) => r.data?.findAudios.count ?? 0,
        getItems: (r) => r.data?.findAudios.audios ?? [],
        filterHook,
      },
    });

  const { filter, setFilter } = filterState;

  const { effectiveFilter, result, cachedResult, items, totalCount } =
    queryResult;

  const {
    selectedIds,
    selectedItems,
    onSelectChange,
    onSelectAll,
    onSelectNone,
    onInvertSelection,
    hasSelection,
  } = listSelect;

  const { modal, showModal, closeModal } = modalState;

  // Utility hooks
  const { setPage, removeCriterion, clearAllCriteria } = useFilterOperations({
    filter,
    setFilter,
  });

  useAddKeybinds(filter, totalCount);
  useFilteredSidebarKeybinds({
    showSidebar,
    setShowSidebar,
  });

  const onCloseEditDelete = useCloseEditDelete({
    closeModal,
    onSelectNone,
    result,
  });

  const onEdit = useCallback(() => {
    showModal(
      <EditAudiosDialog selected={selectedItems} onClose={onCloseEditDelete} />
    );
  }, [showModal, selectedItems, onCloseEditDelete]);

  const onDelete = useCallback(() => {
    showModal(
      <DeleteAudiosDialog
        selected={selectedItems}
        onClose={onCloseEditDelete}
      />
    );
  }, [showModal, selectedItems, onCloseEditDelete]);

  useEffect(() => {
    Mousetrap.bind("e", () => {
      if (hasSelection) {
        onEdit?.();
      }
    });

    Mousetrap.bind("d d", () => {
      if (hasSelection) {
        onDelete?.();
      }
    });

    return () => {
      Mousetrap.unbind("e");
      Mousetrap.unbind("d d");
    };
  }, [hasSelection, onEdit, onDelete]);

  useZoomKeybinds({
    zoomIndex: filter.zoomIndex,
    onChangeZoom: (zoom) => setFilter(filter.setZoom(zoom)),
  });

  const metadataByline = useMemo(() => {
    if (cachedResult.loading) return null;

    return renderMetadataByline(cachedResult) ?? null;
  }, [cachedResult]);

  const queue = useMemo(() => AudioQueue.fromListFilterModel(filter), [filter]);

  const playRandom = usePlayRandom(effectiveFilter, totalCount);
  const playSelected = usePlaySelected(selectedIds);
  const playFirst = usePlayFirst();

  function onCreateNew() {
    history.push("/audios/new");
  }

  function onPlay() {
    if (items.length === 0) {
      return;
    }

    // if there are selected items, play those
    if (hasSelection) {
      playSelected();
      return;
    }

    // otherwise, play the first item in the list
    const audioID = items[0].id;
    playFirst(queue, audioID, 0);
  }

  function onExport(all: boolean) {
    showModal(
      <ExportDialog
        // ExportObjectsInput has no `audios` field yet (pre-existing
        // backend/schema gap — audio export was never wired up server-side,
        // unrelated to this merge). Cast to keep existing UI behavior.
        exportInput={
          {
            audios: {
              ids: Array.from(selectedIds.values()),
              all: all,
            },
          } as GQL.ExportObjectsInput
        }
        onClose={() => closeModal()}
      />
    );
  }

  const otherOperations = [
    {
      text: intl.formatMessage({ id: "actions.play" }),
      onClick: () => onPlay(),
      isDisplayed: () => items.length > 0,
      className: "play-item",
    },
    {
      text: intl.formatMessage(
        { id: "actions.create_entity" },
        { entityType: intl.formatMessage({ id: "audio" }) }
      ),
      onClick: () => onCreateNew(),
      isDisplayed: () => !hasSelection,
      className: "create-new-item",
    },
    {
      text: intl.formatMessage({ id: "actions.select_all" }),
      onClick: () => onSelectAll(),
      isDisplayed: () => totalCount > 0,
    },
    {
      text: intl.formatMessage({ id: "actions.select_none" }),
      onClick: () => onSelectNone(),
      isDisplayed: () => hasSelection,
    },
    {
      text: intl.formatMessage({ id: "actions.invert_selection" }),
      onClick: () => onInvertSelection(),
      isDisplayed: () => totalCount > 0,
    },
    {
      text: intl.formatMessage({ id: "actions.play_random" }),
      onClick: playRandom,
      isDisplayed: () => totalCount > 1,
    },
    {
      text: intl.formatMessage({ id: "actions.export" }),
      onClick: () => onExport(false),
      isDisplayed: () => hasSelection,
    },
    {
      text: intl.formatMessage({ id: "actions.export_all" }),
      onClick: () => onExport(true),
    },
  ];

  // render
  if (sidebarStateLoading) return null;

  const operations = (
    <AudioListOperations
      items={items.length}
      hasSelection={hasSelection}
      operations={otherOperations}
      onEdit={onEdit}
      onDelete={onDelete}
      onPlay={onPlay}
      onCreateNew={onCreateNew}
    />
  );

  return (
    <div
      className={cx("item-list-container audio-list", {
        "hide-sidebar": !showSidebar,
      })}
    >
      {modal}

      <SidebarStateContext.Provider value={{ sectionOpen, setSectionOpen }}>
        <SidebarPane hideSidebar={!showSidebar}>
          <Sidebar hide={!showSidebar} onHide={() => setShowSidebar(false)}>
            <SidebarContent
              filter={filter}
              setFilter={setFilter}
              filterHook={filterHook}
              showEditFilter={showEditFilter}
              view={view}
              sidebarOpen={showSidebar}
              onClose={() => setShowSidebar(false)}
              count={cachedResult.loading ? undefined : totalCount}
              focus={searchFocus}
            />
          </Sidebar>
          <SidebarPaneContent
            onSidebarToggle={() => setShowSidebar(!showSidebar)}
          >
            <FilteredListToolbar
              filter={filter}
              listSelect={listSelect}
              setFilter={setFilter}
              showEditFilter={showEditFilter}
              onDelete={onDelete}
              onEdit={onEdit}
              operationComponent={operations}
              view={view}
              zoomable
            />

            <FilterTags
              view={view}
              criteria={filter.criteria}
              onEditCriterion={(c) => showEditFilter(c.criterionOption.type)}
              onRemoveCriterion={removeCriterion}
              onRemoveAll={clearAllCriteria}
            />

            <div className="pagination-index-container">
              <Pagination
                currentPage={filter.currentPage}
                itemsPerPage={filter.itemsPerPage}
                totalItems={totalCount}
                onChangePage={(page) => setFilter(filter.changePage(page))}
              />
              <PaginationIndex
                loading={cachedResult.loading}
                itemsPerPage={filter.itemsPerPage}
                currentPage={filter.currentPage}
                totalItems={totalCount}
                metadataByline={metadataByline}
              />
            </div>

            <LoadedContent loading={result.loading} error={result.error}>
              <AudioList
                filter={effectiveFilter}
                audios={items}
                selectedIds={selectedIds}
                onSelectChange={onSelectChange}
                fromGroupId={fromGroupId}
              />
            </LoadedContent>

            {totalCount > filter.itemsPerPage && (
              <div className="pagination-footer-container">
                <div className="pagination-footer">
                  <Pagination
                    itemsPerPage={filter.itemsPerPage}
                    currentPage={filter.currentPage}
                    totalItems={totalCount}
                    metadataByline={metadataByline}
                    onChangePage={setPage}
                    pagePopupPlacement="top"
                  />
                </div>
              </div>
            )}
          </SidebarPaneContent>
        </SidebarPane>
      </SidebarStateContext.Provider>
    </div>
  );
};

export default FilteredAudioList;
