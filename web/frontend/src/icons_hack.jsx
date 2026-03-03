// https://github.com/elastic/eui/issues/5463
import { appendIconComponentCache } from '@elastic/eui/es/components/icon/icon';
import { icon as arrowDown } from '@elastic/eui/es/components/icon/assets/arrow_down';
import { icon as arrowUp } from '@elastic/eui/es/components/icon/assets/arrow_up';
import { icon as arrowLeft } from '@elastic/eui/es/components/icon/assets/arrow_left';
import { icon as arrowRight } from '@elastic/eui/es/components/icon/assets/arrow_right';
import { icon as boxesHorizontal } from '@elastic/eui/es/components/icon/assets/boxes_horizontal';
import { icon as check } from '@elastic/eui/es/components/icon/assets/check';
import { icon as checkInCircleFilled } from '@elastic/eui/es/components/icon/assets/checkInCircleFilled';
import { icon as copyClipboard } from '@elastic/eui/es/components/icon/assets/copy_clipboard';
import { icon as cross } from '@elastic/eui/es/components/icon/assets/cross';
import { icon as crossInCircle } from '@elastic/eui/es/components/icon/assets/crossInCircle';
import { icon as minusInCircle } from '@elastic/eui/es/components/icon/assets/minus_in_circle';
import { icon as minusInCircleFilled } from '@elastic/eui/es/components/icon/assets/minus_in_circle_filled';
import { icon as download } from '@elastic/eui/es/components/icon/assets/download';
import { icon as empty } from '@elastic/eui/es/components/icon/assets/empty';
import { icon as eye } from '@elastic/eui/es/components/icon/assets/eye';
import { icon as fullScreen } from '@elastic/eui/es/components/icon/assets/full_screen';
import { icon as gear } from '@elastic/eui/es/components/icon/assets/gear';
import { icon as help } from '@elastic/eui/es/components/icon/assets/help';
import { icon as importIcon } from '@elastic/eui/es/components/icon/assets/import';
import { icon as logoElastic } from '@elastic/eui/es/components/icon/assets/logo_elastic';
import { icon as logoKibana } from '@elastic/eui/es/components/icon/assets/logo_kibana';
import { icon as pencil } from '@elastic/eui/es/components/icon/assets/pencil';
import { icon as play } from '@elastic/eui/es/components/icon/assets/play';
import { icon as plusInCircle } from '@elastic/eui/es/components/icon/assets/plus_in_circle';
import { icon as plusInCircleFilled } from '@elastic/eui/es/components/icon/assets/plus_in_circle_filled';
import { icon as search } from '@elastic/eui/es/components/icon/assets/search';
import { icon as sortable } from '@elastic/eui/es/components/icon/assets/sortable';
import { icon as sortDown } from '@elastic/eui/es/components/icon/assets/sort_down';
import { icon as sortUp } from '@elastic/eui/es/components/icon/assets/sort_up';
import { icon as starFilled } from '@elastic/eui/es/components/icon/assets/star_filled';
import { icon as trash } from '@elastic/eui/es/components/icon/assets/trash';
import { icon as user } from '@elastic/eui/es/components/icon/assets/user';
import { icon as warning } from '@elastic/eui/es/components/icon/assets/warning';
import { icon as alert } from '@elastic/eui/es/components/icon/assets/alert';
import { icon as refresh } from '@elastic/eui/es/components/icon/assets/refresh';

// Register all icons in the component cache for static usage
appendIconComponentCache({
  alert,
  arrowDown,
  arrowLeft,
  arrowRight,
  arrowUp,
  boxesHorizontal,
  check,
  checkInCircleFilled,
  copyClipboard,
  cross,
  crossInCircle,
  minusInCircle,
  minusInCircleFilled,
  download,
  empty,
  eye,
  fullScreen,
  gear,
  help,
  importAction: importIcon,  // EUI uses iconType "importAction", path ./assets/import
  logoElastic,
  logoKibana,
  pencil,
  play,
  plusInCircle,
  plusInCircleFilled,
  refresh,
  search,
  sortable,
  sortDown,
  sortUp,
  starFilled,
  trash,
  user,
  warning,
});
