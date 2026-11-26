<template>
  <div class="card floating">
    <div class="card-title">
      <h2>{{ $t("prompts.fileInfo") }}</h2>
    </div>

    <div class="card-content">
      <p v-if="selected.length > 1">
        {{ $t("prompts.filesSelected", { count: selected.length }) }}
      </p>

      <p class="break-word" v-if="selected.length < 2">
        <strong>{{ $t("prompts.displayName") }}</strong> {{ name }}
      </p>

      <p v-if="!dir || selected.length > 1">
        <strong>{{ $t("prompts.size") }}:</strong>
        <span id="content_length"></span> {{ humanSize }}
      </p>

      <div v-if="resolution">
        <strong>{{ $t("prompts.resolution") }}:</strong>
        {{ resolution.width }} x {{ resolution.height }}
      </div>

      <p v-if="selected.length < 2" :title="modTime">
        <strong>{{ $t("prompts.lastModified") }}:</strong> {{ humanTime }}
      </p>

      <template v-if="dir && selected.length === 0">
        <p>
          <strong>{{ $t("prompts.numberFiles") }}:</strong> {{ req.numFiles }}
        </p>
        <p>
          <strong>{{ $t("prompts.numberDirs") }}:</strong> {{ req.numDirs }}
        </p>
      </template>

      <template v-if="!dir">
        <p>
          <strong>MD5: </strong
          ><code
            ><a
              @click="checksum($event, 'md5')"
              @keypress.enter="checksum($event, 'md5')"
              tabindex="2"
              >{{ $t("prompts.show") }}</a
            ></code
          >
        </p>
        <p>
          <strong>SHA1: </strong
          ><code
            ><a
              @click="checksum($event, 'sha1')"
              @keypress.enter="checksum($event, 'sha1')"
              tabindex="3"
              >{{ $t("prompts.show") }}</a
            ></code
          >
        </p>
        <p>
          <strong>SHA256: </strong
          ><code
            ><a
              @click="checksum($event, 'sha256')"
              @keypress.enter="checksum($event, 'sha256')"
              tabindex="4"
              >{{ $t("prompts.show") }}</a
            ></code
          >
        </p>
        <p>
          <strong>SHA512: </strong
          ><code
            ><a
              @click="checksum($event, 'sha512')"
              @keypress.enter="checksum($event, 'sha512')"
              tabindex="5"
              >{{ $t("prompts.show") }}</a
            ></code
          >
        </p>
      </template>

      <div v-if="metaPath" class="exif-meta">
        <p class="exif-head">
          <strong>{{ $t("prompts.workflow") }}:</strong>
          <span v-if="exif && exif.hasText">{{
            kindLabel || $t("prompts.workflowFound")
          }}</span>
          <span v-else-if="exif">{{ $t("prompts.workflowMissing") }}</span>
          <button
            v-else
            class="button button--flat exif-read"
            :disabled="exifLoading"
            @click="loadExif"
          >
            {{
              exifLoading
                ? $t("prompts.workflowLoading")
                : $t("prompts.workflowRead")
            }}
          </button>
        </p>

        <template v-if="exif && exif.hasText">
          <p v-for="row in summaryRows" :key="row.label" class="break-word">
            <strong>{{ row.label }}:</strong> {{ row.value }}
          </p>

          <p v-if="exif.summary && exif.summary.positive" class="break-word">
            <strong>{{ $t("prompts.positivePrompt") }}:</strong>
            {{ exif.summary.positive }}
          </p>
          <p v-if="exif.summary && exif.summary.negative" class="break-word">
            <strong>{{ $t("prompts.negativePrompt") }}:</strong>
            {{ exif.summary.negative }}
          </p>

          <div v-for="(field, index) in exif.fields" :key="index">
            <p class="exif-field-key">
              <strong>{{ field.key }}</strong>
              <span class="exif-source">{{ field.source }}</span>
              <button
                v-if="field.value.length > 80"
                class="button button--flat exif-copy"
                @click="copyField(field, index)"
              >
                {{
                  copiedIndex === index
                    ? $t("prompts.workflowCopied")
                    : $t("prompts.workflowCopy")
                }}
              </button>
            </p>
            <pre class="break-word">{{ field.value }}</pre>
          </div>
        </template>
      </div>
    </div>

    <div class="card-action">
      <button
        id="focus-prompt"
        type="submit"
        @click="closeHovers"
        class="button button--flat"
        :aria-label="$t('buttons.ok')"
        :title="$t('buttons.ok')"
      >
        {{ $t("buttons.ok") }}
      </button>
    </div>
  </div>
</template>

<script>
import { mapActions, mapState } from "pinia";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";
import { filesize } from "@/utils";
import dayjs from "dayjs";
import { files as api } from "@/api";

const kindLabels = {
  comfyui: "ComfyUI",
  a1111: "A1111 / WebUI",
  nai: "NovelAI",
};

// Reading the metadata can be expensive (large PNGs get decoded for the
// NovelAI alpha payload), so results are only fetched on demand and reused
// while the panel is reopened for the same file.
const exifCache = new Map();

export default {
  name: "info",
  inject: ["$showError"],
  data() {
    return {
      exif: null,
      exifLoading: false,
      copiedIndex: -1,
    };
  },
  watch: {
    metaPath: {
      immediate: true,
      handler(path) {
        this.copiedIndex = -1;
        this.exif = path && exifCache.has(path) ? exifCache.get(path) : null;
      },
    },
  },
  computed: {
    ...mapState(useFileStore, [
      "req",
      "selected",
      "selectedCount",
      "isListing",
    ]),
    humanSize: function () {
      if (this.selectedCount === 0 || !this.isListing) {
        return filesize(this.req.size);
      }

      let sum = 0;

      for (const selected of this.selected) {
        sum += this.req.items[selected].size;
      }

      return filesize(sum);
    },
    humanTime: function () {
      if (this.selectedCount === 0) {
        return dayjs(this.req.modified).fromNow();
      }

      return dayjs(this.req.items[this.selected[0]].modified).fromNow();
    },
    modTime: function () {
      if (this.selectedCount === 0) {
        return new Date(Date.parse(this.req.modified)).toLocaleString();
      }

      return new Date(
        Date.parse(this.req.items[this.selected[0]].modified)
      ).toLocaleString();
    },
    name: function () {
      return this.selectedCount === 0
        ? this.req.name
        : this.req.items[this.selected[0]].name;
    },
    dir: function () {
      return (
        this.selectedCount > 1 ||
        (this.selectedCount === 0
          ? this.req.isDir
          : this.req.items[this.selected[0]].isDir)
      );
    },
    resolution: function () {
      if (this.selectedCount === 1) {
        const selectedItem = this.req.items[this.selected[0]];
        if (selectedItem && selectedItem.type === "image") {
          return selectedItem.resolution;
        }
      } else if (this.req && this.req.type === "image") {
        return this.req.resolution;
      }
      return null;
    },
    metaPath: function () {
      if (this.selectedCount > 0) {
        if (this.selectedCount !== 1) {
          return null;
        }
        const item = this.req.items[this.selected[0]];
        return item && item.type === "image" ? item.path : null;
      }
      return this.req && this.req.type === "image" ? this.req.path : null;
    },
    kindLabel: function () {
      if (!this.exif) {
        return "";
      }
      return kindLabels[this.exif.kind] || this.exif.kind;
    },
    summaryRows: function () {
      const s = this.exif && this.exif.summary;
      if (!s) {
        return [];
      }

      const rows = [];
      if (s.model) {
        rows.push({ label: this.$t("prompts.model"), value: s.model });
      }
      if (s.sampler) {
        rows.push({
          label: this.$t("prompts.sampler"),
          value: s.scheduler ? `${s.sampler} (${s.scheduler})` : s.sampler,
        });
      }
      if (s.steps) {
        rows.push({ label: this.$t("prompts.steps"), value: s.steps });
      }
      if (s.cfg) {
        rows.push({ label: "CFG", value: s.cfg });
      }
      if (s.seed) {
        rows.push({ label: this.$t("prompts.seed"), value: s.seed });
      }
      if (s.width) {
        rows.push({
          label: this.$t("prompts.size"),
          value: `${s.width} x ${s.height}`,
        });
      }
      return rows;
    },
  },
  methods: {
    ...mapActions(useLayoutStore, ["closeHovers"]),
    checksum: async function (event, algo) {
      event.preventDefault();

      let link;

      if (this.selectedCount) {
        link = this.req.items[this.selected[0]].url;
      } else {
        link = this.$route.path;
      }

      try {
        const hash = await api.checksum(link, algo);
        event.target.textContent = hash;
      } catch (e) {
        this.$showError(e);
      }
    },
    async loadExif() {
      const path = this.metaPath;
      if (!path) {
        return;
      }

      this.exifLoading = true;
      try {
        const meta = await api.getExifMeta(path);
        exifCache.set(path, meta);
        this.exif = meta;
      } catch (e) {
        this.$showError(e);
      } finally {
        this.exifLoading = false;
      }
    },
    async copyField(field, index) {
      try {
        await navigator.clipboard.writeText(field.value);
        this.copiedIndex = index;
        setTimeout(() => (this.copiedIndex = -1), 1500);
      } catch (e) {
        this.$showError(e);
      }
    },
  },
};
</script>

<style scoped>
.exif-meta {
  margin-top: 1em;
  padding-top: 0.75em;
  border-top: 1px solid var(--divider);
}

.exif-head {
  display: flex;
  align-items: center;
  gap: 0.5em;
  flex-wrap: wrap;
}

.exif-read {
  padding: 0.1em 0.75em;
  min-width: auto;
}

.exif-read:disabled {
  cursor: default;
  opacity: 0.6;
}

.exif-field-key {
  display: flex;
  align-items: center;
  gap: 0.5em;
  flex-wrap: wrap;
  margin-bottom: 0.25em;
}

.exif-source {
  font-size: 0.75em;
  opacity: 0.6;
}

.exif-copy {
  padding: 0 0.5em;
  min-width: auto;
  font-size: 0.75em;
}

.exif-meta pre {
  max-height: 16em;
  overflow: auto;
  margin: 0 0 0.75em;
  padding: 0.5em;
  font-size: 0.75em;
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--surfaceSecondary);
  border-radius: 4px;
}
</style>
