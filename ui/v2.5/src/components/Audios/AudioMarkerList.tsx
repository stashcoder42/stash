import cloneDeep from "lodash-es/cloneDeep";
import React, { useCallback, useEffect } from "react";
import { useHistory } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import Mousetrap from "mousetrap";
import * as GQL from "src/core/generated-graphql";
import {
  queryFindAudioMarkers,
  useFindAudioMarkers,
} from "src/core/StashService";
import { useFilteredItemList } from "../List/ItemList";
import { ListFilterModel } from "src/models/list-filter/filter";
import { DisplayMode } from "src/models/list-filter/types";
import { AudioMarkerWallPanel } from "./AudioMarkerWallPanel";
import { View } from "../List/views";
import { AudioMarkerCardGrid } from "./AudioMarkerCardGrid";
import { PatchComponent, PatchContainerComponent } from "src/patch";
import {
  FilteredListToolbar,
  IItemListOperation,
} from "../List/FilteredListToolbar";
import {
  Sidebar,
  SidebarPane,
  SidebarPaneContent,
  SidebarStateContext,
  useSidebarState,
} from "../Shared/Sidebar";
import { useFilterOperations } from "../List/util";
import {
  FilteredSidebarHeader,
  useFilteredSidebarKeybinds,
} from "../List/Filters/FilterSidebar";
import {
  IListFilterOperation,
  ListOperations,
} from "../List/ListOperationButtons";
import cx from "classnames";
import { FilterTags } from "../List/FilterTags";
import { Pagination, PaginationIndex } from "../List/Pagination";
import { LoadedContent } from "../List/PagedList";
import useFocus from "src/utils/focus";
import { SidebarTagsFilter } from "../List/Filters/TagsFilter";
import { Button } from "react-bootstrap";

const AudioMarkerListContent: React.FC<{
  markers: GQL.AudioMarkerDataFragment[];
  filter: ListFilterModel;
  selectedIds: Set<string>;
  onSelectChange: (id: string, selected: boolean, shiftKey: boolean) => void;
}> = PatchComponent(
  "AudioMarkerList",
  ({ markers, filter, selectedIds, onSelectChange }) => {
    if (markers.length === 0) {
      return null;
    }

    if (filter.displayMode === DisplayMode.Wall) {
      return (
        <AudioMarkerWallPanel
          markers={markers}
          zoomIndex={filter.zoomIndex}
          selectedIds={selectedIds}
          onSelectChange={onSelectChange}
        />
      );
    }

    if (filter.displayMode === DisplayMode.Grid) {
      return (
        <AudioMarkerCardGrid
          markers={markers}
          zoomIndex={filter.zoomIndex}
          selectedIds={selectedIds}
          onSelectChange={onSelectChange}
        />
      );
    }

    return null;
  }
);

function usePlayRandom(filter: ListFilterModel, count: number) {
  const history = useHistory();

  const playRandom = useCallback(async () => {
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
    const queryResults = await queryFindAudioMarkers(filterCopy);
    const marker = queryResults.data.findAudioMarkers.audio_markers[index];
    if (marker) {
      const url = `/audios/${marker.audio.id}?t=${marker.seconds}`;
      history.push(url);
    }
  }, [filter, count, history]);

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

const AudioMarkersFilterSidebarSections = PatchContainerComponent(
  "FilteredAudioMarkerList.SidebarSections"
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

      <AudioMarkersFilterSidebarSections>
        <SidebarTagsFilter
          filter={filter}
          setFilter={setFilter}
          filterHook={filterHook}
        />
      </AudioMarkersFilterSidebarSections>

      <div className="sidebar-footer">
        <Button className="sidebar-close-button" onClick={onClose}>
          <FormattedMessage id={showResultsId} values={{ count }} />
        </Button>
      </div>
    </>
  );
};

interface IAudioMarkerList {
  filterHook?: (filter: ListFilterModel) => ListFilterModel;
  view?: View;
  alterQuery?: boolean;
  defaultSort?: string;
  extraOperations?: IItemListOperation<GQL.FindAudioMarkersQueryResult>[];
}

export const FilteredAudioMarkerList = PatchComponent(
  "FilteredAudioMarkerList",
  (props: IAudioMarkerList) => {
    const intl = useIntl();

    const searchFocus = useFocus();

    const {
      filterHook,
      defaultSort,
      view,
      alterQuery,
      extraOperations = [],
    } = props;

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
          filterMode: GQL.FilterMode.AudioMarkers,
          defaultSort,
          view,
          useURL: alterQuery,
        },
        queryResultProps: {
          useResult: useFindAudioMarkers,
          getCount: (r) => r.data?.findAudioMarkers.count ?? 0,
          getItems: (r) => r.data?.findAudioMarkers.audio_markers ?? [],
          filterHook,
        },
      });

    const { filter, setFilter } = filterState;

    const { effectiveFilter, result, cachedResult, items, totalCount } =
      queryResult;

    const {
      selectedIds,
      onSelectChange,
      onSelectAll,
      onSelectNone,
      onInvertSelection,
      hasSelection,
    } = listSelect;

    const { modal } = modalState;

    const { setPage, removeCriterion, clearAllCriteria } = useFilterOperations({
      filter,
      setFilter,
    });

    useAddKeybinds(filter, totalCount);
    useFilteredSidebarKeybinds({
      showSidebar,
      setShowSidebar,
    });

    const convertedExtraOperations: IListFilterOperation[] =
      extraOperations.map((o) => ({
        ...o,
        isDisplayed: o.isDisplayed
          ? () => o.isDisplayed!(result, filter, selectedIds)
          : undefined,
        onClick: () => {
          o.onClick(result, filter, selectedIds);
        },
      }));

    const playRandom = usePlayRandom(effectiveFilter, totalCount);

    const otherOperations = [
      ...convertedExtraOperations,
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
    ];

    if (sidebarStateLoading) return null;

    const operations = (
      <ListOperations
        items={items.length}
        hasSelection={hasSelection}
        operations={otherOperations}
        operationsMenuClassName="audio-marker-list-operations-dropdown"
      />
    );

    return (
      <div
        className={cx("item-list-container audio-marker-list", {
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
                operationComponent={operations}
                view={view}
                zoomable
              />

              <FilterTags
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
                />
              </div>

              <LoadedContent loading={result.loading} error={result.error}>
                <AudioMarkerListContent
                  filter={effectiveFilter}
                  markers={items}
                  selectedIds={selectedIds}
                  onSelectChange={onSelectChange}
                />
              </LoadedContent>

              {totalCount > filter.itemsPerPage && (
                <div className="pagination-footer-container">
                  <div className="pagination-footer">
                    <Pagination
                      itemsPerPage={filter.itemsPerPage}
                      currentPage={filter.currentPage}
                      totalItems={totalCount}
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
  }
);

export default FilteredAudioMarkerList;
