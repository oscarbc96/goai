<script setup lang="ts">
import { ref, onMounted } from "vue";

interface Version {
  tag: string;
  date: string;
  label: string;
}

const versions = ref<Version[]>([]);
const current = ref("latest");
const latestTag = ref("");
const branchBaseURL = ref("");

// Detect if we're on a branch deploy by matching CF Pages branch subdomain.
// Branch deploys: v0-4-2.goai-2lz.pages.dev → version "v0.4.2"
function detectBranchVersion(): string | null {
  const host = window.location.hostname;
  const match = host.match(/^(v[\d-]+)\./);
  if (match) {
    return match[1].replace(/-/g, ".");
  }
  return null;
}

onMounted(async () => {
  try {
    // Always fetch from canonical source to get consistent version data.
    const res = await fetch("https://goai.sh/versions.json");
    if (res.ok) {
      const data = await res.json();
      versions.value = data.versions || [];
      latestTag.value = data.latest || "";
      branchBaseURL.value = data.branchBaseURL || "";

      // Detect current version from branch deploy hostname.
      const branchVer = detectBranchVersion();
      if (branchVer) {
        current.value = branchVer;
      }
    }
  } catch {
    // versions.json not available
  }
});

function navigate(tag: string) {
  const path = window.location.pathname;
  if (tag === "latest") {
    window.location.href = "https://goai.sh" + path;
  } else {
    // CF Pages branch names use dashes: v0.4.2 → v0-4-2
    const branch = tag.replace(/\./g, "-");
    const url = branchBaseURL.value.replace("{branch}", branch);
    window.location.href = url + path;
  }
}
</script>

<template>
  <div v-if="versions.length > 1" class="version-switcher">
    <select
      :value="current"
      @change="navigate(($event.target as HTMLSelectElement).value)"
    >
      <option value="latest">
        latest{{ latestTag ? ` (${latestTag})` : "" }}
      </option>
      <option v-for="v in versions" :key="v.tag" :value="v.tag">
        {{ v.tag }}
      </option>
    </select>
  </div>
</template>

<style scoped>
.version-switcher {
  margin-left: 8px;
}

.version-switcher select {
  background: transparent;
  color: var(--vp-c-text-2);
  border: 1px solid var(--vp-c-divider);
  padding: 3px 8px;
  font-family: var(--vp-font-family-mono);
  font-size: 0.72rem;
  font-weight: 500;
  cursor: pointer;
  appearance: none;
  -webkit-appearance: none;
  padding-right: 20px;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6'%3E%3Cpath d='M1 1l4 4 4-4' fill='none' stroke='%23999' stroke-width='1.5'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 6px center;
}

.version-switcher select:hover {
  color: var(--vp-c-text-1);
  border-color: var(--vp-c-text-3);
}
</style>
