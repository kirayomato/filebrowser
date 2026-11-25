<template>
  <div>
    <header-bar showMenu showLogo>
      <search />
      <title />
      <action
        class="search-button"
        icon="search"
        :label="t('buttons.search')"
        @action="openSearch()"
      />

      <template #actions>
        <template v-if="!isMobile">
          <action
            v-if="headerButtons.share"
            icon="share"
            :label="t('buttons.share')"
            show="share"
          />
          <action
            v-if="headerButtons.rename"
            icon="mode_edit"
            :label="t('buttons.rename')"
            show="rename"
          />
          <action
            v-if="headerButtons.copy"
            id="copy-button"
            icon="content_copy"
            :label="t('buttons.copyFile')"
            show="copy"
          />
          <action
            v-if="headerButtons.move"
            id="move-button"
            icon="forward"
            :label="t('buttons.moveFile')"
            show="move"
          />
          <action
            v-if="headerButtons.delete"
            id="delete-button"
            icon="delete"
            :label="t('buttons.delete')"
            show="delete"
          />
        </template>

        <action
          v-if="headerButtons.shell"
          icon="code"
          :label="t('buttons.shell')"
          @action="layoutStore.toggleShell"
        />
        <action
          :icon="viewIcon"
          :label="t('buttons.switchView')"
          @action="switchView"
        />
        <action
          :icon="flatMode ? 'account_tree' : 'view_list'"
          :label="flatMode ? t('buttons.treeView') : t('buttons.flatView')"
          @action="toggleFlat"
        />
        <action
          v-if="headerButtons.download"
          icon="file_download"
          :label="t('buttons.download')"
          @action="download"
          :counter="fileStore.selectedCount"
        />
        <action
          v-if="headerButtons.upload"
          icon="file_upload"
          id="upload-button"
          :label="t('buttons.upload')"
          @action="uploadFunc"
        />
        <action icon="info" :label="t('buttons.info')" show="info" />
        <action
          icon="check_circle"
          :label="t('buttons.selectMultiple')"
          @action="toggleMultipleSelection"
        />
      </template>
    </header-bar>

    <div
      v-if="isMobile"
      id="file-selection"
      :class="{
        'file-selection-margin-bottom': fileStore.multiple,
      }"
    >
      <span v-if="fileStore.selectedCount > 0">
        {{ t("prompts.filesSelected", fileStore.selectedCount) }}
      </span>
      <action
        v-if="headerButtons.share"
        icon="share"
        :label="t('buttons.share')"
        show="share"
      />
      <action
        v-if="headerButtons.rename"
        icon="mode_edit"
        :label="t('buttons.rename')"
        show="rename"
      />
      <action
        v-if="headerButtons.copy"
        icon="content_copy"
        :label="t('buttons.copyFile')"
        show="copy"
      />
      <action
        v-if="headerButtons.move"
        icon="forward"
        :label="t('buttons.moveFile')"
        show="move"
      />
      <action
        v-if="headerButtons.delete"
        icon="delete"
        :label="t('buttons.delete')"
        show="delete"
      />
    </div>

    <div v-if="layoutStore.loading">
      <h2 class="message delayed">
        <div class="spinner">
          <div class="bounce1"></div>
          <div class="bounce2"></div>
          <div class="bounce3"></div>
        </div>
        <span>{{ t("files.loading") }}</span>
      </h2>
    </div>
    <template v-else>
      <div v-if="flatMode" id="flat-listing">
        <div class="flat-filters">
          <input
            v-model.trim="flatFilter.query"
            type="text"
            :placeholder="t('search.search')"
            :aria-label="t('search.search')"
            @keyup.enter="fetchFlat"
          />
          <select
            v-model="flatFilter.type"
            :aria-label="t('files.flat.allTypes')"
          >
            <option value="">{{ t("files.flat.allTypes") }}</option>
            <option value="dir">{{ t("files.folders") }}</option>
            <option value="file">{{ t("files.files") }}</option>
            <option value="image">{{ t("search.images") }}</option>
            <option value="audio">{{ t("search.music") }}</option>
            <option value="video">{{ t("search.video") }}</option>
            <option value="pdf">PDF</option>
          </select>
          <input
            v-model.number="flatFilter.minSize"
            class="flat-size"
            type="number"
            min="0"
            :placeholder="t('files.flat.minSize')"
            :aria-label="t('files.flat.minSize')"
          />
          <span class="flat-dash">-</span>
          <input
            v-model.number="flatFilter.maxSize"
            class="flat-size"
            type="number"
            min="0"
            :placeholder="t('files.flat.maxSize')"
            :aria-label="t('files.flat.maxSize')"
          />
          <select v-model="flatFilter.sizeUnit" aria-label="unit">
            <option value="B">B</option>
            <option value="KB">KB</option>
            <option value="MB">MB</option>
            <option value="GB">GB</option>
          </select>
          <input
            v-model="flatFilter.after"
            type="date"
            :title="t('files.flat.after')"
            :aria-label="t('files.flat.after')"
          />
          <input
            v-model="flatFilter.before"
            type="date"
            :title="t('files.flat.before')"
            :aria-label="t('files.flat.before')"
          />
          <button
            class="action"
            :title="t('buttons.clear')"
            :aria-label="t('buttons.clear')"
            @click="resetFlatFilter"
          >
            <i class="material-icons">clear</i>
          </button>
        </div>

        <div class="item header flat-row">
          <p
            :class="{ active: nameSorted }"
            class="name"
            role="button"
            tabindex="0"
            @click="sort('name')"
            :title="t('files.sortByName')"
          >
            <span>{{ t("files.name") }}</span>
            <i class="material-icons">{{ nameIcon }}</i>
          </p>
          <p
            :class="{ active: pathSorted }"
            class="path"
            role="button"
            tabindex="0"
            @click="sort('path')"
            :title="t('files.flat.path')"
          >
            <span>{{ t("files.flat.path") }}</span>
            <i class="material-icons">{{ pathIcon }}</i>
          </p>
          <p
            :class="{ active: sizeSorted }"
            class="size"
            role="button"
            tabindex="0"
            @click="sort('size')"
            :title="t('files.sortBySize')"
          >
            <span>{{ t("files.size") }}</span>
            <i class="material-icons">{{ sizeIcon }}</i>
          </p>
          <p
            :class="{ active: modifiedSorted }"
            class="modified"
            role="button"
            tabindex="0"
            @click="sort('modified')"
            :title="t('files.sortByLastModified')"
          >
            <span>{{ t("files.lastModified") }}</span>
            <i class="material-icons">{{ modifiedIcon }}</i>
          </p>
        </div>

        <p v-if="flatLoading && flatEntries.length === 0" class="message">
          {{ t("files.loading") }}
        </p>
        <template v-else>
          <div
            v-if="flatEntries.length > 0"
            ref="flatScroll"
            class="flat-scroll"
            @scroll.passive="onFlatScroll"
          >
            <div class="flat-spacer" :style="{ height: `${flatTotalHeight}px` }">
              <div
                class="flat-window"
                :style="{ transform: `translateY(${flatOffsetY}px)` }"
              >
                <router-link
                  v-for="entry in visibleFlatEntries"
                  :key="entry.path"
                  class="item flat-row"
                  :to="flatURL(entry)"
                >
                  <span class="name">
                    <i class="material-icons">{{ flatIcon(entry) }}</i>
                    {{ entry.name }}
                  </span>
                  <span class="path">{{ relativePath(entry) }}</span>
                  <span class="size">
                    {{ entry.isDir ? "—" : filesize(entry.size) }}
                  </span>
                  <span class="modified">
                    <time :datetime="entry.modified">
                      {{ dayjs(entry.modified).format("L LT") }}
                    </time>
                  </span>
                </router-link>
              </div>
            </div>
          </div>
          <p v-if="flatEntries.length === 0" class="message flat-empty">
            {{ t("files.flat.empty") }}
          </p>
        </template>
        <p class="flat-count">{{ flatEntries.length }} {{ t("files.flat.count") }}</p>
      </div>
      <div
        v-else-if="
          (fileStore.req?.numDirs ?? 0) + (fileStore.req?.numFiles ?? 0) == 0
        "
      >
        <h2 class="message">
          <i class="material-icons">sentiment_dissatisfied</i>
          <span>{{ t("files.lonely") }}</span>
        </h2>
        <input
          style="display: none"
          type="file"
          id="upload-input"
          @change="uploadInput($event)"
          multiple
        />
        <input
          style="display: none"
          type="file"
          id="upload-folder-input"
          @change="uploadInput($event)"
          webkitdirectory
          multiple
        />
      </div>
      <div
        v-else
        id="listing"
        ref="listing"
        class="file-icons"
        data-clear-on-click="true"
        :class="authStore.user?.viewMode ?? ''"
        @click="handleEmptyAreaClick"
      >
        <div>
          <div class="item header">
            <div>
              <p
                :class="{ active: nameSorted }"
                class="name"
                role="button"
                tabindex="0"
                @click="sort('name')"
                :title="t('files.sortByName')"
                :aria-label="t('files.sortByName')"
              >
                <span>{{ t("files.name") }}</span>
                <i class="material-icons">{{ nameIcon }}</i>
              </p>

              <p
                :class="{ active: sizeSorted }"
                class="size"
                role="button"
                tabindex="0"
                @click="sort('size')"
                :title="t('files.sortBySize')"
                :aria-label="t('files.sortBySize')"
              >
                <span>{{ t("files.size") }}</span>
                <i class="material-icons">{{ sizeIcon }}</i>
              </p>
              <p
                :class="{ active: modifiedSorted }"
                class="modified"
                role="button"
                tabindex="0"
                @click="sort('modified')"
                :title="t('files.sortByLastModified')"
                :aria-label="t('files.sortByLastModified')"
              >
                <span>{{ t("files.lastModified") }}</span>
                <i class="material-icons">{{ modifiedIcon }}</i>
              </p>
            </div>
          </div>
        </div>

        <h2 data-clear-on-click="true" v-if="fileStore.req?.numDirs ?? false">
          {{ t("files.folders") }}
        </h2>
        <div
          v-if="fileStore.req?.numDirs ?? false"
          data-clear-on-click="true"
          @contextmenu="showContextMenu"
        >
          <item
            v-for="item in dirs"
            :key="base64(item.name)"
            v-bind:index="item.index"
            v-bind:name="item.name"
            v-bind:isDir="item.isDir"
            v-bind:url="item.url"
            v-bind:modified="item.modified"
            v-bind:type="item.type"
            v-bind:size="item.size"
            v-bind:path="item.path"
          >
          </item>
        </div>

        <h2 data-clear-on-click="true" v-if="fileStore.req?.numFiles ?? false">
          {{ t("files.files") }}
        </h2>
        <div
          v-if="fileStore.req?.numFiles ?? false"
          data-clear-on-click="true"
          @contextmenu="showContextMenu"
        >
          <item
            v-for="item in files"
            :key="base64(item.name)"
            v-bind:index="item.index"
            v-bind:name="item.name"
            v-bind:isDir="item.isDir"
            v-bind:url="item.url"
            v-bind:modified="item.modified"
            v-bind:type="item.type"
            v-bind:size="item.size"
            v-bind:path="item.path"
          >
          </item>
        </div>
        <context-menu
          :show="isContextMenuVisible"
          :pos="contextMenuPos"
          @hide="hideContextMenu"
        >
          <action
            v-if="headerButtons.share"
            icon="share"
            :label="t('buttons.share')"
            show="share"
          />
          <action
            v-if="headerButtons.rename"
            icon="mode_edit"
            :label="t('buttons.rename')"
            show="rename"
          />
          <action
            v-if="headerButtons.copy"
            id="copy-button"
            icon="content_copy"
            :label="t('buttons.copyFile')"
            show="copy"
          />
          <action
            v-if="headerButtons.move"
            id="move-button"
            icon="forward"
            :label="t('buttons.moveFile')"
            show="move"
          />
          <action
            v-if="headerButtons.delete"
            id="delete-button"
            icon="delete"
            :label="t('buttons.delete')"
            show="delete"
          />
          <action
            v-if="headerButtons.download"
            icon="file_download"
            :label="t('buttons.download')"
            @action="download"
            :counter="fileStore.selectedCount"
          />
          <action icon="info" :label="t('buttons.info')" show="info" />
        </context-menu>

        <input
          style="display: none"
          type="file"
          id="upload-input"
          @change="uploadInput($event)"
          multiple
        />
        <input
          style="display: none"
          type="file"
          id="upload-folder-input"
          @change="uploadInput($event)"
          webkitdirectory
          multiple
        />

        <div :class="{ active: fileStore.multiple }" id="multiple-selection">
          <p>{{ t("files.multipleSelectionEnabled") }}</p>
          <div
            @click="() => (fileStore.multiple = false)"
            tabindex="0"
            role="button"
            :title="t('buttons.clear')"
            :aria-label="t('buttons.clear')"
            class="action"
          >
            <i class="material-icons">clear</i>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from "@/stores/auth";
import { useClipboardStore } from "@/stores/clipboard";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";

import { users, files as api } from "@/api";
import { enableExec } from "@/utils/constants";
import * as upload from "@/utils/upload";
import buttons from "@/utils/buttons";
import css from "@/utils/css";
import { filesize } from "@/utils";
import dayjs from "dayjs";
import { debounce, throttle } from "lodash-es";
import { Base64 } from "js-base64";
import { encodePath } from "@/utils/url";

import HeaderBar from "@/components/header/HeaderBar.vue";
import Action from "@/components/header/Action.vue";
import Search from "@/components/Search.vue";
import Item from "@/components/files/ListingItem.vue";
import ContextMenu from "@/components/ContextMenu.vue";
import {
  computed,
  inject,
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
} from "vue";
import { useRoute, onBeforeRouteUpdate } from "vue-router";
import { useI18n } from "vue-i18n";
import { storeToRefs } from "pinia";
import { removePrefix } from "@/api/utils";

interface FlatItem {
  entry: RecursiveEntry;
  nameKey: string;
  pathKey: string;
  timeKey: number;
}

const flatMode = ref<boolean>(false);
const flatRaw = ref<FlatItem[]>([]);
const flatEntries = ref<RecursiveEntry[]>([]);
const flatLoading = ref<boolean>(false);
const flatSortBy = ref<string>("name");
const flatAsc = ref<boolean>(true);
const flatFilter = reactive({
  query: "",
  type: "",
  minSize: "" as number | string,
  maxSize: "" as number | string,
  sizeUnit: "MB",
  after: "",
  before: "",
});
let flatController = new AbortController();

const sizeUnits: Record<string, number> = {
  B: 1,
  KB: 1024,
  MB: 1024 * 1024,
  GB: 1024 * 1024 * 1024,
};

const toBytes = (value: number | string, unit: string): number | undefined => {
  if (value === "" || value === null || value === undefined) return undefined;
  const size = Number(value);
  if (!Number.isFinite(size) || size < 0) return undefined;
  return Math.floor(size * (sizeUnits[unit] ?? 1));
};

const naturalKey = (value: string): string =>
  value.toLowerCase().replace(/\d+/g, (digits) =>
    String(Number(digits)).padStart(12, "0")
  );

const compareKey = (a: string, b: string): number =>
  a < b ? -1 : a > b ? 1 : 0;

const compareFlat = (a: FlatItem, b: FlatItem): number => {
  if (a.entry.isDir !== b.entry.isDir) return a.entry.isDir ? -1 : 1;

  let result = 0;
  switch (flatSortBy.value) {
    case "size":
      result = a.entry.size - b.entry.size;
      break;
    case "modified":
      result = a.timeKey - b.timeKey;
      break;
    case "path":
      result = compareKey(a.pathKey, b.pathKey);
      break;
    default:
      result = compareKey(a.nameKey, b.nameKey);
  }
  if (result === 0) result = compareKey(a.pathKey, b.pathKey);
  return flatAsc.value ? result : -result;
};

const applyFlatSort = () => {
  flatEntries.value = [...flatRaw.value].sort(compareFlat).map((item) => item.entry);
};

const flushFlat = throttle(applyFlatSort, 100);

const flatRowHeight = 40;
const flatOverscan = 6;
const flatScroll = ref<HTMLElement | null>(null);
const flatScrollTop = ref(0);
const flatViewportHeight = ref(window.innerHeight - 220);

const onFlatScroll = (event: Event) => {
  flatScrollTop.value = (event.target as HTMLElement).scrollTop;
};

const flatStartIndex = computed(() =>
  Math.max(0, Math.floor(flatScrollTop.value / flatRowHeight) - flatOverscan)
);

const visibleFlatEntries = computed(() => {
  const count =
    Math.ceil(flatViewportHeight.value / flatRowHeight) + flatOverscan * 2;
  return flatEntries.value.slice(flatStartIndex.value, flatStartIndex.value + count);
});

const flatOffsetY = computed(() => flatStartIndex.value * flatRowHeight);
const flatTotalHeight = computed(() => flatEntries.value.length * flatRowHeight);

const fetchFlat = async () => {
  flatController.abort();
  const controller = new AbortController();
  flatController = controller;
  flushFlat.cancel();
  flatRaw.value = [];
  flatEntries.value = [];
  flatScrollTop.value = 0;
  flatLoading.value = true;

  try {
    await api.fetchAllStream(
      route.path,
      {
        query: flatFilter.query,
        type: flatFilter.type,
        minSize: toBytes(flatFilter.minSize, flatFilter.sizeUnit),
        maxSize: toBytes(flatFilter.maxSize, flatFilter.sizeUnit),
        after: flatFilter.after,
        before: flatFilter.before,
      },
      (entry) => {
        if (flatController !== controller) return;
        flatRaw.value.push({
          entry,
          nameKey: naturalKey(entry.name),
          pathKey: naturalKey(entry.path),
          timeKey: Date.parse(entry.modified),
        });
        flushFlat();
      },
      controller.signal
    );
    if (flatController === controller) applyFlatSort();
  } catch (e: any) {
    if (e?.is_canceled) return;
    $showError(e);
  } finally {
    if (flatController === controller) flatLoading.value = false;
  }
};

const debouncedFetchFlat = debounce(fetchFlat, 300);

const toggleFlat = () => {
  flatMode.value = !flatMode.value;
  if (flatMode.value) {
    fetchFlat();
  } else {
    flatController.abort();
  }
};

const resetFlatFilter = () => {
  flatFilter.query = "";
  flatFilter.type = "";
  flatFilter.minSize = "";
  flatFilter.maxSize = "";
  flatFilter.sizeUnit = "MB";
  flatFilter.after = "";
  flatFilter.before = "";
};

const flatSort = (by: string) => {
  if (flatSortBy.value === by) {
    flatAsc.value = !flatAsc.value;
    return;
  }
  flatSortBy.value = by;
  flatAsc.value = by === "name" || by === "path";
};

const flatIcon = (entry: RecursiveEntry) => {
  if (entry.isDir) return "folder";
  const extension = entry.extension.toLowerCase();
  if ([".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg"].includes(extension)) {
    return "insert_photo";
  }
  if ([".mp4", ".mkv", ".avi", ".mov", ".webm"].includes(extension)) {
    return "movie";
  }
  if ([".mp3", ".flac", ".wav", ".ogg", ".m4a"].includes(extension)) {
    return "volume_up";
  }
  if (extension === ".pdf") return "picture_as_pdf";
  return "insert_drive_file";
};

const flatURL = (entry: RecursiveEntry) =>
  "/files" + encodePath(entry.path) + (entry.isDir ? "/" : "");

const relativePath = (entry: RecursiveEntry) => {
  const root = removePrefix(route.path);
  const prefix = root.endsWith("/") ? root : root + "/";
  return entry.path.startsWith(prefix) ? entry.path.slice(prefix.length) : entry.path;
};

const showLimit = ref<number>(50);
const columnWidth = ref<number>(280);
const dragCounter = ref<number>(0);
const width = ref<number>(window.innerWidth);
const itemWeight = ref<number>(0);
const isContextMenuVisible = ref<boolean>(false);
const contextMenuPos = ref<{ x: number; y: number }>({ x: 0, y: 0 });

const $showError = inject<IToastError>("$showError")!;

const clipboardStore = useClipboardStore();
const authStore = useAuthStore();
const fileStore = useFileStore();
const layoutStore = useLayoutStore();

const { req } = storeToRefs(fileStore);

const route = useRoute();
onBeforeRouteUpdate(() => {
  hideContextMenu();
});

watch(flatFilter, () => {
  if (flatMode.value) debouncedFetchFlat();
}, { deep: true });

watch(() => route.path, () => {
  if (flatMode.value) fetchFlat();
});

const { t } = useI18n();

const listing = ref<HTMLElement | null>(null);

const nameSorted = computed(() =>
  flatMode.value
    ? flatSortBy.value === "name"
    : fileStore.req
      ? fileStore.req.sorting.by === "name"
      : false
);

const sizeSorted = computed(() =>
  flatMode.value
    ? flatSortBy.value === "size"
    : fileStore.req
      ? fileStore.req.sorting.by === "size"
      : false
);

const modifiedSorted = computed(() =>
  flatMode.value
    ? flatSortBy.value === "modified"
    : fileStore.req
      ? fileStore.req.sorting.by === "modified"
      : false
);

const pathSorted = computed(
  () => flatMode.value && flatSortBy.value === "path"
);

const ascOrdered = computed(() =>
  flatMode.value
    ? flatAsc.value
    : fileStore.req
      ? fileStore.req.sorting.asc
      : false
);

const dirs = computed(() => items.value.dirs.slice(0, showLimit.value));

const items = computed(() => {
  const dirs: any[] = [];
  const files: any[] = [];

  fileStore.req?.items.forEach((item) => {
    if (item.isDir) {
      dirs.push(item);
    } else {
      files.push(item);
    }
  });

  return { dirs, files };
});

const files = computed((): Resource[] => {
  let _showLimit = showLimit.value - items.value.dirs.length;

  if (_showLimit < 0) _showLimit = 0;

  return items.value.files.slice(0, _showLimit);
});

const nameIcon = computed(() => {
  if (nameSorted.value && !ascOrdered.value) {
    return "arrow_upward";
  }

  return "arrow_downward";
});

const sizeIcon = computed(() => {
  if (sizeSorted.value && ascOrdered.value) {
    return "arrow_downward";
  }

  return "arrow_upward";
});

const modifiedIcon = computed(() => {
  if (modifiedSorted.value && ascOrdered.value) {
    return "arrow_downward";
  }

  return "arrow_upward";
});

const pathIcon = computed(() => {
  if (pathSorted.value && !ascOrdered.value) {
    return "arrow_upward";
  }

  return "arrow_downward";
});

const viewIcon = computed(() => {
  const icons = {
    list: "view_module",
    mosaic: "grid_view",
    "mosaic gallery": "view_list",
  };
  return authStore.user === null
    ? icons["list"]
    : icons[authStore.user.viewMode];
});

const headerButtons = computed(() => {
  return {
    upload: authStore.user?.perm.create,
    download: authStore.user?.perm.download,
    shell: authStore.user?.perm.execute && enableExec,
    delete: fileStore.selectedCount > 0 && authStore.user?.perm.delete,
    rename: fileStore.selectedCount === 1 && authStore.user?.perm.rename,
    share:
      fileStore.selectedCount === 1 &&
      authStore.user?.perm.share &&
      authStore.user?.perm.download,
    move: fileStore.selectedCount > 0 && authStore.user?.perm.rename,
    copy: fileStore.selectedCount > 0 && authStore.user?.perm.create,
  };
});

const isMobile = computed(() => {
  return width.value <= 736;
});

watch(req, () => {
  // Reset the show value
  showLimit.value = 50;

  nextTick(() => {
    // Ensures that the listing is displayed
    // How much every listing item affects the window height
    setItemWeight();

    // Scroll to the item opened previously
    if (!revealPreviousItem()) {
      // Fill and fit the window with listing items
      fillWindow(true);
    }
  });
});

onMounted(() => {
  // Check the columns size for the first time.
  columnsResize();

  // How much every listing item affects the window height
  setItemWeight();

  // Scroll to the item opened previously
  if (!revealPreviousItem()) {
    // Fill and fit the window with listing items
    fillWindow(true);
  }

  // Add the needed event listeners to the window and document.
  window.addEventListener("keydown", keyEvent);
  window.addEventListener("scroll", scrollEvent);
  window.addEventListener("resize", windowsResize);

  if (!authStore.user?.perm.create) return;
  document.addEventListener("dragover", preventDefault);
  document.addEventListener("dragenter", dragEnter);
  document.addEventListener("dragleave", dragLeave);
  document.addEventListener("drop", drop);
});

onBeforeUnmount(() => {
  debouncedFetchFlat.cancel();
  flushFlat.cancel();
  flatController.abort();

  // Remove event listeners before destroying this page.
  window.removeEventListener("keydown", keyEvent);
  window.removeEventListener("scroll", scrollEvent);
  window.removeEventListener("resize", windowsResize);

  if (authStore.user && !authStore.user?.perm.create) return;
  document.removeEventListener("dragover", preventDefault);
  document.removeEventListener("dragenter", dragEnter);
  document.removeEventListener("dragleave", dragLeave);
  document.removeEventListener("drop", drop);
});

const base64 = (name: string) => Base64.encodeURI(name);

const keyEvent = (event: KeyboardEvent) => {
  // No prompts are shown
  if (layoutStore.currentPrompt !== null) {
    return;
  }

  if (event.key === "Escape") {
    // Reset files selection.
    fileStore.selected = [];
  }

  if (event.key === "Delete") {
    if (!authStore.user?.perm.delete || fileStore.selectedCount == 0) return;

    // Show delete prompt.
    layoutStore.showHover("delete");
  }

  if (event.key === "F2") {
    if (!authStore.user?.perm.rename || fileStore.selectedCount !== 1) return;

    // Show rename prompt.
    layoutStore.showHover("rename");
  }

  // Ctrl is pressed
  if (!event.ctrlKey && !event.metaKey) {
    return;
  }

  switch (event.key) {
    case "f":
    case "F":
      if (event.shiftKey) {
        event.preventDefault();
        layoutStore.showHover("search");
      }
      break;
    case "c":
    case "x":
      copyCut(event);
      break;
    case "v":
      paste(event);
      break;
    case "a":
      event.preventDefault();
      for (const file of items.value.files) {
        if (fileStore.selected.indexOf(file.index) === -1) {
          fileStore.selected.push(file.index);
        }
      }
      for (const dir of items.value.dirs) {
        if (fileStore.selected.indexOf(dir.index) === -1) {
          fileStore.selected.push(dir.index);
        }
      }
      break;
    case "s":
      event.preventDefault();
      document.getElementById("download-button")?.click();
      break;
  }
};

const preventDefault = (event: Event) => {
  // Wrapper around prevent default.
  event.preventDefault();
};

const copyCut = (event: Event | KeyboardEvent): void => {
  if ((event.target as HTMLElement).tagName?.toLowerCase() === "input") return;

  if (fileStore.req === null) return;

  const items = [];

  for (const i of fileStore.selected) {
    items.push({
      from: fileStore.req.items[i].url,
      name: fileStore.req.items[i].name,
      size: fileStore.req.items[i].size,
      isDir: fileStore.req.items[i].isDir,
      modified: fileStore.req.items[i].modified,
    });
  }

  if (items.length === 0) {
    return;
  }

  clipboardStore.$patch({
    key: (event as KeyboardEvent).key,
    items,
    path: route.path,
  });
};

const paste = async (event: Event) => {
  if ((event.target as HTMLElement).tagName?.toLowerCase() === "input") return;

  // TODO router location should it be
  const items: any[] = [];

  for (const item of clipboardStore.items) {
    const from = item.from.endsWith("/") ? item.from.slice(0, -1) : item.from;
    const to = route.path + encodeURIComponent(item.name);
    items.push({
      from,
      to,
      name: item.name,
      size: item.size,
      isDir: item.isDir,
      modified: item.modified,
      overwrite: false,
      rename: clipboardStore.path == route.path,
    });
  }

  if (items.length === 0) {
    return;
  }

  const preselect = removePrefix(route.path) + items[0].name;

  let action = (overwrite?: boolean, rename?: boolean) => {
    api
      .copy(items, overwrite, rename)
      .then(() => {
        fileStore.preselect = preselect;
        fileStore.reload = true;
      })
      .catch($showError);
  };

  if (clipboardStore.key === "x") {
    action = (overwrite, rename) => {
      api
        .move(items, overwrite, rename)
        .then(() => {
          clipboardStore.resetClipboard();
          fileStore.preselect = preselect;
          fileStore.reload = true;
        })
        .catch($showError);
    };
  }

  const path = route.path.endsWith("/") ? route.path : route.path + "/";
  const conflict = await upload.checkConflict(items, path, true);

  if (conflict.length > 0) {
    layoutStore.showHover({
      prompt: "resolve-conflict",
      props: {
        conflict: conflict,
      },
      confirm: (event: Event, result: Array<ConflictingResource>) => {
        event.preventDefault();
        layoutStore.closeHovers();
        for (let i = result.length - 1; i >= 0; i--) {
          const item = result[i];
          if (item.checked.length == 2) {
            items[item.index].rename = true;
          } else if (item.checked.length == 1 && item.checked[0] == "origin") {
            items[item.index].overwrite = true;
          } else {
            items.splice(item.index, 1);
          }
        }
        if (items.length > 0) {
          action();
        }
      },
    });

    return;
  }

  action(false, false);
};

const columnsResize = () => {
  // Update the columns size based on the window width.
  const items_ = css(["#listing.mosaic .item", ".mosaic#listing .item"]);
  if (items_ === null) return;

  let columns = Math.floor(
    (document.querySelector("main")?.offsetWidth ?? 0) / columnWidth.value
  );
  if (columns === 0) columns = 1;
  items_.style.width = `calc(${100 / columns}% - 1em)`;
};

const scrollEvent = throttle(() => {
  const totalItems =
    (fileStore.req?.numDirs ?? 0) + (fileStore.req?.numFiles ?? 0);

  // All items are displayed
  if (showLimit.value >= totalItems) return;

  const currentPos = window.innerHeight + window.scrollY;

  // Trigger at the 75% of the window height
  const triggerPos = document.body.offsetHeight - window.innerHeight * 0.25;

  if (currentPos > triggerPos) {
    // Quantity of items needed to fill 2x of the window height
    const showQuantity = Math.ceil((window.innerHeight * 2) / itemWeight.value);

    // Increase the number of displayed items
    showLimit.value += showQuantity;
  }
}, 100);

const dragEnter = () => {
  dragCounter.value++;

  // When the user starts dragging an item, put every
  // file on the listing with 50% opacity.
  const items = document.getElementsByClassName("item");

  Array.from(items).forEach((file: Element) => {
    (file as HTMLElement).style.opacity = "0.5";
  });
};

const dragLeave = () => {
  dragCounter.value--;

  if (dragCounter.value == 0) {
    resetOpacity();
  }
};

const drop = async (event: DragEvent) => {
  event.preventDefault();
  dragCounter.value = 0;
  resetOpacity();

  const dt = event.dataTransfer;
  let el: HTMLElement | null = event.target as HTMLElement;

  if (fileStore.req === null || dt === null || dt.files.length <= 0) return;

  for (let i = 0; i < 5; i++) {
    if (el !== null && !el.classList.contains("item")) {
      el = el.parentElement;
    }
  }

  const files: UploadList = (await upload.scanFiles(dt)) as UploadList;
  let path = route.path.endsWith("/") ? route.path : route.path + "/";

  if (
    el !== null &&
    el.classList.contains("item") &&
    el.dataset.dir === "true"
  ) {
    // Get url from ListingItem instance
    // TODO: Don't know what is happening here
    path = el.__vue__.url;

    try {
      (await api.fetch(path)).items;
    } catch (error: any) {
      $showError(error);
      return;
    }
  }

  // Checking the destination hits the server, so show it is working rather
  // than leaving the action looking inert until the upload starts.
  buttons.loading("upload");
  const conflict = await upload.checkConflict(files, path);

  const preselect = removePrefix(path) + (files[0].fullPath || files[0].name);

  if (conflict.length > 0) {
    buttons.done("upload");
    layoutStore.showHover({
      prompt: "resolve-conflict",
      props: {
        conflict: conflict,
        isUploadAction: true,
      },
      confirm: (event: Event, result: Array<ConflictingResource>) => {
        event.preventDefault();
        layoutStore.closeHovers();
        for (let i = result.length - 1; i >= 0; i--) {
          const item = result[i];
          if (item.checked.length == 2) {
            continue;
          } else if (item.checked.length == 1 && item.checked[0] == "origin") {
            files[item.index].overwrite = true;
          } else {
            files.splice(item.index, 1);
          }
        }
        if (files.length > 0) {
          upload.handleFiles(files, path);
          fileStore.preselect = preselect;
        }
      },
    });

    return;
  }

  upload.handleFiles(files, path);
  fileStore.preselect = preselect;
};

const uploadInput = async (event: Event) => {
  const files = (event.currentTarget as HTMLInputElement)?.files;
  if (files === null) return;

  const folder_upload = !!files[0].webkitRelativePath;

  const uploadFiles: UploadList = [];
  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    const fullPath = folder_upload ? file.webkitRelativePath : undefined;
    uploadFiles.push({
      file,
      name: file.name,
      size: file.size,
      isDir: false,
      fullPath,
    });
  }

  const path = route.path.endsWith("/") ? route.path : route.path + "/";

  // Checking the destination hits the server, so show it is working rather
  // than leaving the action looking inert until the upload starts.
  buttons.loading("upload");
  const conflict = await upload.checkConflict(uploadFiles, path);

  if (conflict.length > 0) {
    buttons.done("upload");
    layoutStore.showHover({
      prompt: "resolve-conflict",
      props: {
        conflict: conflict,
        isUploadAction: true,
      },
      confirm: (event: Event, result: Array<ConflictingResource>) => {
        event.preventDefault();
        layoutStore.closeHovers();
        for (let i = result.length - 1; i >= 0; i--) {
          const item = result[i];
          if (item.checked.length == 2) {
            continue;
          } else if (item.checked.length == 1 && item.checked[0] == "origin") {
            uploadFiles[item.index].overwrite = true;
          } else {
            uploadFiles.splice(item.index, 1);
          }
        }
        if (uploadFiles.length > 0) {
          upload.handleFiles(uploadFiles, path);
        }
      },
    });

    return;
  }

  upload.handleFiles(uploadFiles, path);
};

const resetOpacity = () => {
  const items = document.getElementsByClassName("item");

  Array.from(items).forEach((file: Element) => {
    (file as HTMLElement).style.opacity = "1";
  });
};

const sort = async (by: string) => {
  if (flatMode.value) {
    flatSort(by);
    applyFlatSort();
    flatScrollTop.value = 0;
    if (flatScroll.value) flatScroll.value.scrollTop = 0;
    return;
  }

  let asc = false;

  if (by === "name") {
    if (nameIcon.value === "arrow_upward") {
      asc = true;
    }
  } else if (by === "size") {
    if (sizeIcon.value === "arrow_upward") {
      asc = true;
    }
  } else if (by === "modified") {
    if (modifiedIcon.value === "arrow_upward") {
      asc = true;
    }
  }

  try {
    if (authStore.user?.id) {
      await users.update({ id: authStore.user?.id, sorting: { by, asc } }, [
        "sorting",
      ]);
    }
  } catch (e: any) {
    $showError(e);
  }

  fileStore.reload = true;
};

const openSearch = () => {
  layoutStore.showHover("search");
};

const toggleMultipleSelection = () => {
  fileStore.toggleMultiple();
  layoutStore.closeHovers();
};

const windowsResize = throttle(() => {
  columnsResize();
  width.value = window.innerWidth;
  flatViewportHeight.value = window.innerHeight - 220;

  // Listing element is not displayed
  if (listing.value == null) return;

  // How much every listing item affects the window height
  setItemWeight();

  // Fill but not fit the window
  fillWindow();
}, 100);

const download = () => {
  if (fileStore.req === null) return;

  if (
    fileStore.selectedCount === 1 &&
    !fileStore.req.items[fileStore.selected[0]].isDir
  ) {
    api.download(null, fileStore.req.items[fileStore.selected[0]].url);
    return;
  }

  layoutStore.showHover({
    prompt: "download",
    confirm: (format: any) => {
      layoutStore.closeHovers();

      const files = [];

      if (fileStore.selectedCount > 0 && fileStore.req !== null) {
        for (const i of fileStore.selected) {
          files.push(fileStore.req.items[i].url);
        }
      } else {
        files.push(route.path);
      }

      api.download(format, ...files);
    },
  });
};

const switchView = async () => {
  layoutStore.closeHovers();

  const modes = {
    list: "mosaic",
    mosaic: "mosaic gallery",
    "mosaic gallery": "list",
  };

  const data = {
    id: authStore.user?.id,
    viewMode: (modes[authStore.user?.viewMode ?? "list"] ||
      "list") as ViewModeType,
  };

  users.update(data, ["viewMode"]).catch($showError);

  authStore.updateUser(data);

  setItemWeight();
  fillWindow();
};

const uploadFunc = () => {
  if (
    typeof window.DataTransferItem !== "undefined" &&
    typeof DataTransferItem.prototype.webkitGetAsEntry !== "undefined"
  ) {
    layoutStore.showHover("upload");
  } else {
    document.getElementById("upload-input")?.click();
  }
};

const setItemWeight = () => {
  // Listing element is not displayed
  if (listing.value === null || fileStore.req === null) return;

  let itemQuantity = fileStore.req.numDirs + fileStore.req.numFiles;
  if (itemQuantity > showLimit.value) itemQuantity = showLimit.value;

  // How much every listing item affects the window height
  itemWeight.value = listing.value.offsetHeight / itemQuantity;
};

const fillWindow = (fit = false) => {
  if (fileStore.req === null) return;

  const totalItems = fileStore.req.numDirs + fileStore.req.numFiles;

  // More items are displayed than the total
  if (showLimit.value >= totalItems && !fit) return;

  const windowHeight = window.innerHeight;

  // Quantity of items needed to fill 2x of the window height
  const showQuantity = Math.ceil(
    (windowHeight + windowHeight * 2) / itemWeight.value
  );

  // Less items to display than current
  if (showLimit.value > showQuantity && !fit) return;

  // Set the number of displayed items
  showLimit.value = showQuantity > totalItems ? totalItems : showQuantity;
};

const revealPreviousItem = () => {
  if (!fileStore.req || !fileStore.oldReq) return;

  const index = fileStore.selected[0];
  if (index === undefined) return;

  showLimit.value =
    index + Math.ceil((window.innerHeight * 2) / itemWeight.value);

  nextTick(() => {
    const items = document.querySelectorAll("#listing .item");
    items[index].scrollIntoView({ block: "center" });
  });

  return true;
};

const showContextMenu = (event: MouseEvent) => {
  event.preventDefault();
  isContextMenuVisible.value = true;
  contextMenuPos.value = {
    x: event.clientX + 8,
    y: event.clientY + Math.floor(window.scrollY),
  };
};

const hideContextMenu = () => {
  isContextMenuVisible.value = false;
};

const handleEmptyAreaClick = (e: MouseEvent) => {
  const target = e.target;
  if (!(target instanceof HTMLElement)) return;

  if (target.dataset.clearOnClick === "true") {
    fileStore.selected = [];
  }
};
</script>
<style scoped>
#listing {
  min-height: calc(100vh - 8rem);
}

.file-selection-margin-bottom {
  margin-bottom: 3.5rem;
}

#flat-listing {
  padding: 0.5em 0 1em;
}

.flat-filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5em;
  margin-bottom: 0.75em;
}

.flat-filters input,
.flat-filters select {
  background: var(--surfacePrimary);
  border: 1px solid var(--borderPrimary);
  border-radius: 4px;
  color: var(--textPrimary);
  padding: 0.4em 0.6em;
}

.flat-filters .flat-size {
  width: 7em;
}

.flat-dash {
  color: var(--textPrimary);
}

.flat-scroll {
  max-height: calc(100vh - 220px);
  overflow-y: auto;
}

.flat-spacer {
  position: relative;
}

.flat-window {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
}

.flat-row {
  display: grid !important;
  grid-template-columns: minmax(10em, 2fr) minmax(10em, 2fr) 7em 12em;
  gap: 0.5em;
  align-items: center;
  height: 40px;
  box-sizing: border-box;
  padding: 0 0.75em;
  border-bottom: 1px solid var(--divider);
  color: var(--textPrimary);
  text-decoration: none;
}

.flat-row.header {
  font-weight: 500;
  color: var(--textSecondary);
  background: var(--surfaceSecondary);
}

.flat-row p {
  display: flex;
  align-items: center;
  gap: 0.25em;
  margin: 0;
  cursor: pointer;
}

.flat-row p.active {
  color: var(--iconPrimary);
}

.flat-row p i {
  font-size: 1.2em;
  margin: 0;
}

.flat-row > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.flat-row > .name {
  font-weight: 500;
}

.flat-row > .name i {
  font-size: 1.5em;
  color: var(--iconPrimary);
}

.flat-row > .path,
.flat-row > .size,
.flat-row > .modified {
  font-size: 0.9em;
}

.flat-row:not(.header):hover {
  background: var(--hover);
}

.flat-count {
  margin: 0.75em 0.5em;
  color: var(--textPrimary);
  font-size: 0.9em;
}

.flat-empty {
  font-size: 1em;
}

@media (max-width: 736px) {
  .flat-row {
    grid-template-columns: minmax(8em, 1fr) 6em;
  }

  .flat-row > .path,
  .flat-row > .modified {
    display: none;
  }
}
</style>
