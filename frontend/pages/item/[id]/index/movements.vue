<script setup lang="ts">
  import type { EntityOut } from "~~/lib/api/types/data-contracts";
  import BaseContainer from "@/components/Base/Container.vue";
  import StockMovementListView from "~/components/StockMovement/ListView.vue";

  const props = defineProps<{
    item: EntityOut;
  }>();
</script>

<template>
  <BaseContainer class="flex flex-col gap-8">
    <StockMovementListView :item-id="props.item.id" :current-quantity="props.item.quantity" />
  </BaseContainer>
</template>
