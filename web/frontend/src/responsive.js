import { useIsWithinBreakpoints } from '@elastic/eui';

// Matches EuiBasicTable's default responsiveBreakpoint ('m'): tables collapse into cards below the 'm'
// breakpoint (current breakpoint xs or s), so mobile-only columns and compact layouts switch on at exactly
// the same width.
export function useIsMobile() {
  return useIsWithinBreakpoints(['xs', 's']);
}
