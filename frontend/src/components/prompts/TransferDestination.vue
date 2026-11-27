<template>
  <div class="card floating">
    <div class="card-title">
      <h2>{{ $t(titleKey) }}</h2>
    </div>

    <div class="card-content">
      <p>{{ $t(messageKey, { count, dest: displayDest }) }}</p>
    </div>

    <div class="card-action">
      <button
        class="button button--flat button--grey"
        @click="stay"
        :aria-label="$t('buttons.stayHere')"
        :title="$t('buttons.stayHere')"
        tabindex="2"
      >
        {{ $t("buttons.stayHere") }}
      </button>
      <button
        ref="primaryButton"
        id="focus-prompt"
        class="button button--flat"
        @click="goToDestination"
        :aria-label="$t('buttons.goToDestination')"
        :title="$t('buttons.goToDestination')"
        tabindex="1"
      >
        {{ $t("buttons.goToDestination") }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, ref } from "vue";
import { useRouter } from "vue-router";
import { useLayoutStore } from "@/stores/layout";

const router = useRouter();
const layoutStore = useLayoutStore();

const primaryButton = ref<HTMLButtonElement | null>(null);

onActivated(() => {
  primaryButton.value?.focus();
});

const promptProps = computed(() => layoutStore.currentPrompt?.props ?? {});
const dest = computed<string>(() => promptProps.value.dest ?? "");
const count = computed<number>(() => promptProps.value.count ?? 1);
const operation = computed<string>(
  () => promptProps.value.operation ?? "move"
);

const titleKey = computed(() =>
  operation.value === "copy"
    ? "prompts.copyComplete"
    : "prompts.moveComplete"
);

const messageKey = computed(() =>
  operation.value === "copy"
    ? "prompts.copyDestinationMessage"
    : "prompts.moveDestinationMessage"
);

const displayDest = computed(() => {
  try {
    return decodeURIComponent(dest.value);
  } catch {
    return dest.value;
  }
});

const stay = () => {
  layoutStore.closeHovers();
};

const goToDestination = () => {
  const prompt = layoutStore.currentPrompt;
  prompt?.confirm?.();
  layoutStore.closeHovers();
  router.push({ path: dest.value });
};
</script>

<style scoped>
.card-content p {
  overflow-wrap: anywhere;
}
</style>
