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
        <action
          v-if="!flatMode"
          icon="info"
          :label="t('buttons.info')"
          show="info"
        />
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
      <div
        v-if="flatMode"
        id="flat-listing"
        :style="{ '--flat-thumb-size': `${flatThumbSize}px` }"
      >
        <div v-if="flatPickerOpen" class="flat-picker">
          <p class="message">{{ t("files.flat.pickFolder") }}</p>
          <div v-if="flatFolders.length > 0" class="flat-picker-list">
            <label
              v-for="folder in flatFolders"
              :key="folder.path"
              class="flat-picker-item"
            >
              <input
                type="checkbox"
                :checked="isFlatPicked('/files' + encodePath(folder.path))"
                @change="toggleFlatPicker('/files' + encodePath(folder.path))"
              />
              <i class="material-icons">folder</i>
              {{ folder.name }}
            </label>
          </div>
          <p v-else class="message">{{ t("files.flat.noFolders") }}</p>
          <div class="flat-picker-actions">
            <button
              v-if="flatFolderPaths.length > 0"
              class="flat-picker-link"
              @click="toggleFlatPickerAll"
            >
              {{
                flatPickerFoldersAll
                  ? t("files.flat.unselectAll")
                  : t("files.flat.selectAll")
              }}
            </button>
            <button
              class="flat-picker-start"
              :disabled="flatPicker.length === 0"
              @click="startFlatScan"
            >
              {{ t("files.flat.startScan") }}
            </button>
            <button class="flat-picker-cancel" @click="cancelFlatPicker">
              {{ t("buttons.cancel") }}
            </button>
          </div>
        </div>
        <div class="flat-filters">
          <input
            v-model.trim="flatFilter.query"
            type="text"
            :placeholder="t('search.search')"
            :aria-label="t('search.search')"
            @keyup.enter="fetchFlat"
          />
          <select
            v-if="enableThumbs"
            v-model="flatThumbSize"
            :aria-label="t('files.flat.thumbSize')"
          >
            <option v-for="size in flatThumbSizes" :key="size" :value="size">
              {{ size }}
            </option>
          </select>
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
          <button
            class="action"
            :title="t('files.flat.changeFolder')"
            :aria-label="t('files.flat.changeFolder')"
            @click="changeFlatFolder"
          >
            <i class="material-icons">folder_open</i>
          </button>
          <button
            v-if="flatDeep"
            class="action"
            :title="t('files.flat.shallow')"
            :aria-label="t('files.flat.shallow')"
            @click="showFlatShallow"
          >
            <i class="material-icons">unfold_less</i>
          </button>
        </div>

        <div class="item header flat-row">
          <label class="flat-check">
            <input
              type="checkbox"
              :checked="flatAllSelected"
              :aria-label="t('files.flat.selectAll')"
              @change="toggleFlatSelectAll"
            />
          </label>
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
                  :class="{ 'flat-selected': isFlatSelected(entry) }"
                  :to="flatURL(entry)"
                  :target="entry.isDir ? undefined : '_blank'"
                >
                  <label class="flat-check" @click.stop>
                    <input
                      type="checkbox"
                      :checked="isFlatSelected(entry)"
                      @change="toggleFlatSelected(entry)"
                    />
                  </label>
                  <span class="name">
                    <img
                      v-if="enableThumbs && isFlatImage(entry)"
                      v-lazy="flatThumbnailUrl(entry)"
                      :alt="entry.name"
                    />
                    <i v-else class="material-icons">{{ flatIcon(entry) }}</i>
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
        <p class="flat-count">
          {{ flatEntries.length }} {{ t("files.flat.count") }}
          <span v-if="flatDeep" class="flat-scope">
            · {{ t("files.flat.recursive") }}
          </span>
        </p>
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
          <action
          v-if="!flatMode"
          icon="info"
          :label="t('buttons.info')"
          show="info"
        />
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
import { enableExec, enableThumbs } from "@/utils/constants";
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
  shallowRef,
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

// Kept at module level so leaving the listing for a preview and coming back
// restores the same view without another full walk.
const flatMode = ref<boolean>(false);
const flatRaw = shallowRef<FlatItem[]>([]);
const flatEntries = shallowRef<RecursiveEntry[]>([]);
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
const flatThumbSize = ref<number>(40);
// false 表示只列根目录下一层，true 表示递归整棵子树。
const flatDeep = ref<boolean>(false);
const flatPickerOpen = ref<boolean>(false);
const flatScrollTop = ref(0);
const flatScanRoots = shallowRef<string[]>([]);
const flatSelected = shallowRef<Set<string>>(new Set());
const flatCache = new Map<string, FlatItem[]>();
let flatController = new AbortController();

const flatCacheKey = (roots: string[]): string =>
  JSON.stringify([
    [...roots].sort(),
    flatDeep.value,
    flatFilter.query,
    flatFilter.type,
    toBytes(flatFilter.minSize, flatFilter.sizeUnit),
    toBytes(flatFilter.maxSize, flatFilter.sizeUnit),
    flatFilter.after,
    flatFilter.before,
  ]);

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

let pendingFlat: FlatItem[] = [];

const flushPending = throttle(() => {
  if (pendingFlat.length === 0) return;
  flatRaw.value = flatRaw.value.concat(pendingFlat);
  pendingFlat = [];
  applyFlatSort();
}, 1000);

const flatThumbSizes = [24, 40, 64];
const flatRowHeight = computed(() => flatThumbSize.value + 16);
const flatOverscan = 6;
const flatScroll = ref<HTMLElement | null>(null);
const flatViewportHeight = ref(window.innerHeight - 220);

const onFlatScroll = (event: Event) => {
  flatScrollTop.value = (event.target as HTMLElement).scrollTop;
};

const flatStartIndex = computed(() =>
  Math.max(0, Math.floor(flatScrollTop.value / flatRowHeight.value) - flatOverscan)
);

const visibleFlatEntries = computed(() => {
  const count =
    Math.ceil(flatViewportHeight.value / flatRowHeight.value) + flatOverscan * 2;
  return flatEntries.value.slice(flatStartIndex.value, flatStartIndex.value + count);
});

const flatOffsetY = computed(() => flatStartIndex.value * flatRowHeight.value);
const flatTotalHeight = computed(() => flatEntries.value.length * flatRowHeight.value);

const fetchFlat = async () => {
  const roots = dedupeRoots(flatScanRoots.value);
  if (roots.length === 0) return;
  flatScanRoots.value = roots;

  flatController.abort();
  const controller = new AbortController();
  flatController = controller;
  flushPending.cancel();
  pendingFlat = [];

  const key = flatCacheKey(roots);
  const cached = flatCache.get(key);
  if (cached) {
    flatRaw.value = cached;
    flatScrollTop.value = 0;
    applyFlatSort();
    flatLoading.value = false;
    return;
  }

  flatRaw.value = [];
  flatEntries.value = [];
  flatScrollTop.value = 0;
  flatLoading.value = true;
  clearFlatSelection();

  const params = {
    query: flatFilter.query,
    type: flatFilter.type,
    minSize: toBytes(flatFilter.minSize, flatFilter.sizeUnit),
    maxSize: toBytes(flatFilter.maxSize, flatFilter.sizeUnit),
    after: flatFilter.after,
    before: flatFilter.before,
    depth: flatDeep.value ? undefined : 1,
  };

  const results = await Promise.allSettled(
    roots.map((root) =>
      api.fetchAllStream(
        root,
        params,
        (entry) => {
          if (flatController !== controller) return;
          pendingFlat.push({
            entry,
            nameKey: naturalKey(entry.name),
            pathKey: naturalKey(entry.path),
            timeKey: Date.parse(entry.modified),
          });
          flushPending();
        },
        controller.signal
      )
    )
  );

  try {
    if (flatController !== controller) return;

    if (pendingFlat.length > 0) {
      flatRaw.value = flatRaw.value.concat(pendingFlat);
      pendingFlat = [];
    }
    applyFlatSort();

    const failures = roots.flatMap((root, index) => {
      const result = results[index];
      if (result.status !== "rejected") return [];
      const reason = result.reason;
      if (reason?.is_canceled) return [];
      return [{ root, reason }];
    });

    // 部分根失败时不写缓存，下次进来仍会重试失败的根。
    if (failures.length === 0) flatCache.set(key, flatRaw.value);
    else {
      $showError(
        failures
          .map(
            ({ root, reason }) =>
              `${flatRootLabel(root)}: ${reason?.message ?? reason}`
          )
          .join("; ")
      );
    }
  } finally {
    if (flatController === controller) flatLoading.value = false;
  }
};

const debouncedFetchFlat = debounce(fetchFlat, 300);

const flatResourceUrl = (entry: RecursiveEntry): string =>
  "/files" + encodePath(entry.path);

const syncFlatSelection = () => {
  fileStore.flatSelected = flatEntries.value
    .filter((entry) => flatSelected.value.has(entry.path))
    .map((entry) => ({
      url: flatResourceUrl(entry),
      name: entry.name,
      size: entry.size,
      isDir: entry.isDir,
      modified: entry.modified,
      path: entry.path,
    }));
};

const clearFlatSelection = () => {
  flatSelected.value = new Set();
  fileStore.flatSelected = [];
};

const isFlatSelected = (entry: RecursiveEntry): boolean =>
  flatSelected.value.has(entry.path);

const toggleFlatSelected = (entry: RecursiveEntry) => {
  const next = new Set(flatSelected.value);
  if (next.has(entry.path)) next.delete(entry.path);
  else next.add(entry.path);
  flatSelected.value = next;
  syncFlatSelection();
};

const flatAllSelected = computed(
  () =>
    flatEntries.value.length > 0 &&
    flatEntries.value.every((entry) => flatSelected.value.has(entry.path))
);

const toggleFlatSelectAll = () => {
  flatSelected.value = flatAllSelected.value
    ? new Set()
    : new Set(flatEntries.value.map((entry) => entry.path));
  syncFlatSelection();
};

const flatPicker = ref<string[]>([]);

const isFlatPicked = (path: string): boolean => flatPicker.value.includes(path);

const toggleFlatPicker = (path: string) => {
  const next = new Set(flatPicker.value);
  if (next.has(path)) next.delete(path);
  else next.add(path);
  flatPicker.value = [...next];
};

// 面包屑链接总以 "/" 结尾，扫描根必须归一化，否则请求、缓存键与深度语义都会漂移。
const normalizeRoot = (path: string): string =>
  path.length > 1 && path.endsWith("/") ? path.slice(0, -1) : path;

// 父根已覆盖子根时只保留父根，避免同一子树被多次遍历。
const dedupeRoots = (roots: string[]): string[] => {
  const normalized = [...new Set(roots)].map(normalizeRoot);
  return normalized.filter(
    (root) =>
      !normalized.some((other) => other !== root && root.startsWith(other + "/"))
  );
};

const toggleFlat = () => {
  flatMode.value = !flatMode.value;
  clearFlatSelection();
  fileStore.selected = [];
  flatPickerOpen.value = false;
  flatPicker.value = [];
  if (flatMode.value) {
    // 进入平铺只列当前目录一层，递归扫描必须手动触发。
    flatScanRoots.value = [normalizeRoot(route.path)];
    flatDeep.value = false;
    fetchFlat();
  } else {
    flatController.abort();
  }
};

const startFlatScan = () => {
  const roots = dedupeRoots(flatPicker.value);
  if (roots.length === 0) return;
  flatScanRoots.value = roots;
  flatDeep.value = true;
  flatPicker.value = [];
  flatPickerOpen.value = false;
  fetchFlat();
};

const changeFlatFolder = () => {
  flatPickerOpen.value = !flatPickerOpen.value;
  flatPicker.value = [];
};

const cancelFlatPicker = () => {
  flatPickerOpen.value = false;
  flatPicker.value = [];
};

// 退出递归扫描，回到只列当前目录一层。
const showFlatShallow = () => {
  flatScanRoots.value = [normalizeRoot(route.path)];
  flatDeep.value = false;
  fetchFlat();
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

const isFlatImage = (entry: RecursiveEntry): boolean => entry.type === "image";

const flatThumbnailUrl = (entry: RecursiveEntry): string =>
  api.getPreviewURL(
    { path: entry.path, modified: entry.modified } as Resource,
    "thumb"
  );

const flatIcon = (entry: RecursiveEntry) => {
  switch (entry.type) {
    case "dir":
      return "folder";
    case "image":
      return "insert_photo";
    case "video":
      return "movie";
    case "audio":
      return "volume_up";
    case "pdf":
      return "picture_as_pdf";
    default:
      return "insert_drive_file";
  }
};

const flatURL = (entry: RecursiveEntry) =>
  "/files" + encodePath(entry.path) + (entry.isDir ? "/" : "");

const flatRootLabel = (root: string): string => {
  const parts = removePrefix(root).split("/").filter(Boolean);
  return decodeURIComponent(parts[parts.length - 1] ?? "") || "/";
};

const relativePath = (entry: RecursiveEntry) => {
  const roots = flatScanRoots.value.length ? flatScanRoots.value : [route.path];
  const root =
    roots.find((item) => {
      const base = removePrefix(item);
      return base === "/" || entry.path === base || entry.path.startsWith(base + "/");
    }) ?? roots[0];
  const base = removePrefix(root);
  const rel =
    base === "/"
      ? entry.path.slice(1)
      : entry.path.startsWith(base + "/")
        ? entry.path.slice(base.length + 1)
        : entry.path;
  return roots.length <= 1 ? rel : `${flatRootLabel(root)}/${rel}`;
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

const { req, reload } = storeToRefs(fileStore);

const route = useRoute();
onBeforeRouteUpdate(() => {
  hideContextMenu();
});

const flatFolders = computed(() =>
  (fileStore.req?.items ?? []).filter((item) => item.isDir)
);

const flatFolderPaths = computed(() =>
  flatFolders.value.map((folder) => "/files" + encodePath(folder.path))
);

// 「全选」只作用于子文件夹；「扫描当前目录」包含全部子目录，混在一起会被父根合并掉。
const flatPickerFoldersAll = computed(
  () =>
    flatFolderPaths.value.length > 0 &&
    flatFolderPaths.value.every((path) => flatPicker.value.includes(path))
);

const toggleFlatPickerAll = () => {
  const next = new Set(flatPicker.value);
  if (flatPickerFoldersAll.value) {
    for (const path of flatFolderPaths.value) next.delete(path);
  } else {
    for (const path of flatFolderPaths.value) next.add(path);
  }
  flatPicker.value = [...next];
};

watch(flatFilter, () => {
  if (flatMode.value && flatScanRoots.value.length > 0) debouncedFetchFlat();
}, { deep: true });

watch(() => route.path, () => {
  if (!flatMode.value) return;
  clearFlatSelection();
  flatPicker.value = [];
  flatPickerOpen.value = false;
  flatScanRoots.value = [normalizeRoot(route.path)];
  flatDeep.value = false;
  fetchFlat();
});

watch(reload, (value) => {
  if (!value) return;
  // 多根扫描的缓存键不再是单一路径前缀，整体失效更稳妥。
  flatCache.clear();
  if (flatMode.value && flatScanRoots.value.length > 0) fetchFlat();
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
      !flatMode.value &&
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
  if (flatMode.value) {
    // 组件重建（例如从预览返回）时状态是模块级保留的，只有「当前目录浅层」
    // 这一种情况可以原样恢复，其余一律回到当前目录一层。
    const sameShallowView =
      !flatDeep.value &&
      flatScanRoots.value.length === 1 &&
      flatScanRoots.value[0] === normalizeRoot(route.path);
    if (!sameShallowView) {
      flatScanRoots.value = [normalizeRoot(route.path)];
      flatDeep.value = false;
      flatPickerOpen.value = false;
      flatPicker.value = [];
      fetchFlat();
    }
  }

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
  flushPending.cancel();
  pendingFlat = [];
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
    clearFlatSelection();
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
      if (flatMode.value) {
        flatSelected.value = new Set(
          flatEntries.value.map((entry) => entry.path)
        );
        syncFlatSelection();
        break;
      }
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

  for (const entry of fileStore.selectedEntries) {
    items.push({
      from: entry.url,
      name: entry.name,
      size: entry.size,
      isDir: entry.isDir,
      modified: entry.modified,
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

  const entries = fileStore.selectedEntries;

  if (entries.length === 1 && !entries[0].isDir) {
    api.download(null, entries[0].url);
    return;
  }

  layoutStore.showHover({
    prompt: "download",
    confirm: (format: any) => {
      layoutStore.closeHovers();

      const files = [];

      if (entries.length > 0) {
        for (const entry of entries) {
          files.push(entry.url);
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

.flat-picker {
  max-width: 64em;
}

.flat-picker-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5em;
  margin-bottom: 0.75em;
}

.flat-picker-item {
  display: flex;
  align-items: center;
  gap: 0.4em;
  max-width: 100%;
  padding: 0.5em 0.8em;
  background: var(--surfacePrimary);
  border: 1px solid var(--borderPrimary);
  border-radius: 4px;
  color: var(--textPrimary);
  cursor: pointer;
}

.flat-picker-item:hover {
  border-color: var(--blue);
}

.flat-picker-item i {
  color: var(--blue);
}

.flat-picker-actions {
  display: flex;
  align-items: center;
  gap: 0.5em;
}

.flat-picker-link {
  margin-right: auto;
  padding: 0.5em 0.4em;
  border: none;
  background: transparent;
  color: var(--blue);
  cursor: pointer;
}

.flat-picker-start {
  padding: 0.5em 1.2em;
  border: 1px solid var(--blue);
  border-radius: 4px;
  background: var(--blue);
  color: #fff;
  cursor: pointer;
}

.flat-picker-start:disabled {
  background: transparent;
  border-color: var(--borderPrimary);
  color: var(--textSecondary);
  cursor: default;
}

.flat-picker-cancel {
  padding: 0.5em 1.2em;
  border: 1px solid var(--borderPrimary);
  border-radius: 4px;
  background: transparent;
  color: var(--textPrimary);
  cursor: pointer;
}

.flat-scope {
  color: var(--textSecondary);
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
  grid-template-columns: 2em minmax(10em, 2fr) minmax(10em, 2fr) 7em 12em;
  gap: 0.5em;
  align-items: center;
  height: calc(var(--flat-thumb-size, 40px) + 16px);
  box-sizing: border-box;
  padding: 0 0.75em;
  border-bottom: 1px solid var(--divider);
  color: var(--textPrimary);
  text-decoration: none;
}

.flat-row.flat-selected {
  background: var(--surfaceSecondary);
}

.flat-check {
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.flat-check input {
  margin: 0;
  cursor: pointer;
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

.flat-row > .name img {
  width: var(--flat-thumb-size, 40px);
  height: var(--flat-thumb-size, 40px);
  object-fit: cover;
  margin-right: 0.4em;
  vertical-align: middle;
  border-radius: 2px;
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
