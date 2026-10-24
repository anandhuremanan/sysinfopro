<script lang="ts">
	import { formatBytes } from '$lib';
	import { answerRating, rating } from '$lib/rating.svelte';
	import StrokeIcon from './StrokeIcon.svelte';
</script>

{#if rating.show}
	<div
		role="dialog"
		aria-label="Rate SysInfo Pro"
		class="absolute bottom-4 right-4 z-40 w-[300px] flex gap-2.5 p-3 rounded-xl bg-tile border border-accent/40 shadow-2xl"
	>
		<span class="shrink-0 w-7 h-7 rounded-lg flex items-center justify-center bg-accent/15 text-accent">
			<StrokeIcon d="M12 3l2.7 5.6 6.1.9-4.4 4.3 1 6.1L12 17l-5.4 2.9 1-6.1-4.4-4.3 6.1-.9z" size={15} />
		</span>
		<div class="flex-1 min-w-0">
			<div class="font-semibold text-ink">
				SysInfo Pro has freed {formatBytes(rating.freedBytes)} for you
			</div>
			<div class="text-muted leading-snug mt-0.5">
				If it helped, a rating in the Microsoft Store helps other people find it.
			</div>
			<div class="flex gap-2 mt-2.5">
				<button class="btn btn-primary h-6 px-2.5" onclick={() => answerRating(true)}>Rate it</button>
				<button class="btn btn-secondary h-6 px-2.5" onclick={() => answerRating(false)}>No thanks</button>
			</div>
		</div>
	</div>
{/if}
