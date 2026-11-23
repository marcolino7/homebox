<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import type { StockMovement, StockMovementCreate } from "~~/lib/api/types/data-contracts";
  import { Button } from "@/components/ui/button";
  import { Badge } from "@/components/ui/badge";
  import { Input } from "@/components/ui/input";
  import BaseCard from "@/components/Base/Card.vue";
  import BaseSectionHeader from "@/components/Base/SectionHeader.vue";
  import DateTime from "~/components/global/DateTime.vue";
  import { toast } from "@/components/ui/sonner";

  const props = defineProps<{
    itemId: string;
    currentQuantity?: number;
  }>();

  const api = useUserApi();
  const { t } = useI18n();

  const { data: movements, refresh } = useAsyncData(`stock-movements-${props.itemId}`, async () => {
    const { data, error } = await api.items.stockMovements.getLog(props.itemId);
    if (error) {
      toast.error(t("stock_movements.toast.failed_load"));
      return [] as StockMovement[];
    }
    return data;
  });

  const form = reactive({
    type: "out" as StockMovementCreate["type"],
    quantity: 1,
    comment: "",
  });
  const saving = ref(false);

  const quantityNow = computed(() => {
    const latest = movements.value?.find(m => m.quantityAfter !== undefined && m.quantityAfter !== null);
    return latest?.quantityAfter ?? props.currentQuantity ?? 0;
  });

  async function submit() {
    if (saving.value) return;
    saving.value = true;
    const { error } = await api.items.stockMovements.create(props.itemId, {
      type: form.type,
      quantity: Number(form.quantity),
      comment: form.comment,
    });
    saving.value = false;
    if (error) {
      toast.error(t("stock_movements.toast.failed_create"));
      return;
    }
    toast.success(t("stock_movements.toast.created"));
    form.comment = "";
    await refresh();
    try {
      await refreshNuxtData();
    } catch {
      // the movements list is already up to date
    }
  }

  function typeLabel(type: string) {
    return t(`stock_movements.type.${type}`);
  }

  function signed(n: number) {
    return n > 0 ? `+${n}` : `${n}`;
  }
</script>

<template>
  <section class="space-y-6">
    <BaseCard>
      <BaseSectionHeader class="border-b p-6">
        {{ $t("stock_movements.new") }}
        <template #description>{{ $t("stock_movements.current_quantity") }}: {{ quantityNow }}</template>
      </BaseSectionHeader>
      <form class="flex flex-wrap items-end gap-4 p-6" @submit.prevent="submit">
        <label class="flex flex-col gap-1 text-sm">
          {{ $t("stock_movements.field.type") }}
          <select v-model="form.type" class="h-10 rounded-md border bg-background px-3">
            <option value="in">{{ $t("stock_movements.type.in") }}</option>
            <option value="out">{{ $t("stock_movements.type.out") }}</option>
            <option value="adjustment">{{ $t("stock_movements.type.adjustment") }}</option>
          </select>
        </label>
        <label class="flex flex-col gap-1 text-sm">
          {{ form.type === "adjustment" ? $t("stock_movements.field.new_quantity") : $t("stock_movements.field.quantity") }}
          <Input v-model.number="form.quantity" type="number" min="0" step="any" class="w-32" />
        </label>
        <label class="flex min-w-[12rem] flex-1 flex-col gap-1 text-sm">
          {{ $t("stock_movements.field.comment") }}
          <Input v-model="form.comment" type="text" maxlength="500" />
        </label>
        <Button type="submit" :disabled="saving">{{ $t("stock_movements.add") }}</Button>
      </form>
    </BaseCard>

    <p v-if="!movements || movements.length === 0" class="text-center text-sm text-muted-foreground">
      {{ $t("stock_movements.empty") }}
    </p>

    <BaseCard v-for="m in movements" :key="m.id">
      <div class="flex flex-wrap items-center gap-3 p-4">
        <Badge :variant="m.quantity < 0 ? 'destructive' : 'default'">{{ typeLabel(m.type) }}</Badge>
        <span class="text-lg font-semibold tabular-nums">{{ signed(m.quantity) }}</span>
        <span v-if="m.quantityAfter !== undefined && m.quantityAfter !== null" class="text-sm text-muted-foreground">
          &rarr; {{ m.quantityAfter }}
        </span>
        <span class="text-sm"><DateTime :date="m.occurredAt" format="human" datetime-type="time" /></span>
        <span v-if="m.performedBy" class="text-sm text-muted-foreground">{{ m.performedBy }}</span>
        <span v-if="m.comment" class="text-sm">{{ m.comment }}</span>
      </div>
    </BaseCard>
  </section>
</template>
