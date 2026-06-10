// Computes the next expanded match id when a card is clicked.
// Clicking the currently-expanded card collapses it (returns null);
// clicking any other card opens that one.
export function nextExpandedId(
  current: string | null,
  clicked: string,
): string | null {
  return current === clicked ? null : clicked;
}
