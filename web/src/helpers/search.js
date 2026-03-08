/**
 * Parse a search string into include and exclude terms.
 * Space-separated = AND, `-` prefix = exclude.
 * Example: "gpt -mini -realtime" → { includes: ["gpt"], excludes: ["mini", "realtime"] }
 */
export function parseSearchTerms(searchString) {
  const trimmed = (searchString || '').trim();
  if (!trimmed) return { includes: [], excludes: [] };

  const tokens = trimmed.toLowerCase().split(/\s+/);
  const includes = [];
  const excludes = [];

  for (const token of tokens) {
    if (token.startsWith('-') && token.length > 1) {
      excludes.push(token.slice(1));
    } else {
      includes.push(token);
    }
  }

  return { includes, excludes };
}

/**
 * Check if value(s) match the parsed search terms.
 * Every include term must appear in at least one value (AND).
 * No exclude term may appear in any value.
 * Returns true when terms are empty (no filter).
 */
export function matchesSearchTerms(values, terms) {
  if (!terms || (terms.includes.length === 0 && terms.excludes.length === 0)) {
    return true;
  }

  const valuesArr = (Array.isArray(values) ? values : [values])
    .filter(Boolean)
    .map((v) => v.toLowerCase());

  for (const exc of terms.excludes) {
    if (valuesArr.some((v) => v.includes(exc))) return false;
  }

  for (const inc of terms.includes) {
    if (!valuesArr.some((v) => v.includes(inc))) return false;
  }

  return true;
}
