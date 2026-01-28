import React from "react";
import * as GQL from "src/core/generated-graphql";
import { usePerformerFilterHook } from "src/core/performers";
import { View } from "src/components/List/views";
import { FilteredAudioList } from "src/components/Audios/AudioList";
import { PatchComponent } from "src/patch";

interface IPerformerAudiosProps {
  active: boolean;
  performer: GQL.PerformerDataFragment;
}

export const PerformerAudiosPanel: React.FC<IPerformerAudiosProps> =
  PatchComponent("PerformerAudiosPanel", ({ active, performer }) => {
    const filterHook = usePerformerFilterHook(performer);
    return (
      <FilteredAudioList
        filterHook={filterHook}
        alterQuery={active}
        view={View.PerformerAudios}
      />
    );
  });
