import { useIsWithinBreakpoints } from '@elastic/eui';

// Matches EuiBasicTable's default responsiveBreakpoint ('s'): below the 'm' breakpoint tables collapse
// into cards, so mobile-only columns and compact layouts switch on at exactly the same width.
export function useIsMobile() {
  return useIsWithinBreakpoints(['xs', 's']);
}
