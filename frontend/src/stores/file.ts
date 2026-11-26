import { defineStore } from "pinia";

export interface SelectedEntry {
  url: string;
  name: string;
  size: number;
  isDir: boolean;
  modified: string;
  path: string;
}

export const useFileStore = defineStore("file", {
  // convert to a function
  state: (): {
    req: Resource | null;
    oldReq: Resource | null;
    reload: boolean;
    selected: number[];
    flatSelected: SelectedEntry[];
    multiple: boolean;
    isFiles: boolean;
    preselect: string | null;
  } => ({
    req: null,
    oldReq: null,
    reload: false,
    selected: [],
    flatSelected: [],
    multiple: false,
    isFiles: false,
    preselect: null,
  }),
  getters: {
    selectedCount: (state) =>
      state.flatSelected.length || state.selected.length,
    selectedEntries(state): SelectedEntry[] {
      if (state.flatSelected.length) return state.flatSelected;
      if (!state.req?.items) return [];
      return state.selected
        .map((i) => state.req!.items[i])
        .filter((item): item is ResourceItem => Boolean(item))
        .map((item) => ({
          url: item.url,
          name: item.name,
          size: item.size,
          isDir: item.isDir,
          modified: item.modified,
          path: item.path,
        }));
    },
    // route: () => {
    //   const routerStore = useRouterStore();
    //   return routerStore.router.currentRoute;
    // },
    // isFiles: (state) => {
    //   const layoutStore = useLayoutStore();
    //   return !layoutStore.loading && state.route._value.name === "Files";
    // },
    isListing: (state) => {
      return state.isFiles && state?.req?.isDir;
    },
  },
  actions: {
    // no context as first argument, use `this` instead
    toggleMultiple() {
      this.multiple = !this.multiple;
    },
    updateRequest(value: Resource | null) {
      const selectedItems = this.selected.map((i) => this.req?.items[i]);
      this.oldReq = this.req;
      this.req = value;

      this.selected = [];

      if (!this.req?.items) return;
      this.selected = this.req.items
        .filter((item) =>
          selectedItems.some((rItem) => rItem?.url === item.url)
        )
        .map((item) => item.index);
    },
    removeSelected(value: any) {
      const i = this.selected.indexOf(value);
      if (i === -1) return;
      this.selected.splice(i, 1);
    },
    clearSelection() {
      this.selected = [];
      this.flatSelected = [];
    },
    // easily reset state using `$reset`
    clearFile() {
      this.$reset();
    },
  },
});
