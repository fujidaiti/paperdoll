import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:paperdoll/core/router/routes.dart';
import 'package:paperdoll/debug_keys.dart';
import 'package:paperdoll/features/auth/presentation/providers/auth_providers.dart';
import 'package:paperdoll/features/auth/presentation/sign_in_screen.dart';
import 'package:paperdoll/features/auth/presentation/sign_up_screen.dart';
import 'package:paperdoll/features/auth/presentation/splash_screen.dart';
import 'package:paperdoll/features/auth/presentation/verify_email_screen.dart';
import 'package:paperdoll/features/feed/domain/feed_candidate.dart';
import 'package:paperdoll/features/feed/presentation/attribute_picker_screen.dart';
import 'package:paperdoll/features/feed/presentation/feed_detail_screen.dart';
import 'package:paperdoll/features/feed/presentation/feed_search_screen.dart';
import 'package:paperdoll/features/feed/presentation/feed_subscription_screen.dart';
import 'package:paperdoll/features/feed/presentation/feeds_screen.dart';
import 'package:paperdoll/features/feed_entry/presentation/feed_entry_reader_screen.dart';
import 'package:paperdoll/features/newspaper/presentation/today_screen.dart';
import 'package:paperdoll/features/reading_list/presentation/archived_reading_list_screen.dart';
import 'package:paperdoll/features/reading_list/presentation/reading_list_screen.dart';
import 'package:paperdoll/features/settings/presentation/settings_screen.dart';
import 'package:paperdoll/features/web_clip/presentation/web_clip_reader_screen.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'app_router.g.dart';

final _rootNavigatorKey = GlobalKey<NavigatorState>();

/// Routes a signed-in user has no business being on: reaching one sends them
/// to Today.
const Set<String> _authRoutePaths = {
  routeSplashPath,
  routeSignInPath,
  routeSignUpPath,
  routeVerifyEmailPath,
};

/// Routes a signed-out user may stay on. Verifying an email happens before a
/// token exists, so that route has to be here too, otherwise the screen is
/// bounced to Sign-in the moment it opens.
const Set<String> _signedOutRoutePaths = {
  routeSignInPath,
  routeSignUpPath,
  routeVerifyEmailPath,
};

/// The app's navigation graph: a bottom-nav shell over Today and Feeds, with
/// detail/discovery screens nested under each branch. The feed entry and web
/// article readers push onto the root navigator so they cover the bottom nav
/// bar. Sign-in/up and a splash screen sit outside the shell, gated by
/// [authSessionProvider] via [_authRedirect].
@riverpod
GoRouter goRouter(Ref ref) {
  final sessionAsync = ref.watch(authSessionProvider);
  return GoRouter(
    navigatorKey: _rootNavigatorKey,
    initialLocation: routeSplashPath,
    // The pending attempt is read, not watched: it changes during the flow,
    // and rebuilding the router on every change would tear down the screen
    // the user is on.
    redirect: (context, state) =>
        _authRedirect(sessionAsync, state, ref.read(signUpFlowProvider)),
    routes: [
      GoRoute(
        path: routeSplashPath,
        name: routeSplashName,
        builder: (context, state) => const SplashScreen(),
      ),
      GoRoute(
        path: routeSignInPath,
        name: routeSignInName,
        builder: (context, state) => const SignInScreen(),
      ),
      GoRoute(
        path: routeSignUpPath,
        name: routeSignUpName,
        builder: (context, state) => const SignUpScreen(),
      ),
      GoRoute(
        path: routeVerifyEmailPath,
        name: routeVerifyEmailName,
        builder: (context, state) => const VerifyEmailScreen(),
      ),
      GoRoute(
        path: routeSettingsPath,
        name: routeSettingsName,
        builder: (context, state) => const SettingsScreen(),
      ),
      StatefulShellRoute.indexedStack(
        builder: (context, state, navigationShell) =>
            _ScaffoldWithNavBar(navigationShell: navigationShell),
        branches: [
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: routeTodayPath,
                name: routeTodayName,
                builder: (context, state) => const TodayScreen(),
                routes: [
                  GoRoute(
                    path: routeTodayFeedEntryReaderPath,
                    name: routeTodayFeedEntryReaderName,
                    parentNavigatorKey: _rootNavigatorKey,
                    builder: (context, state) => FeedEntryReaderScreen(
                      id: _idParam(state, 'feedEntryId'),
                    ),
                  ),
                ],
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: routeReadingListPath,
                name: routeReadingListName,
                builder: (context, state) => const ReadingListScreen(),
                routes: [
                  GoRoute(
                    path: routeArchivedReadingListPath,
                    name: routeArchivedReadingListName,
                    builder: (context, state) =>
                        const ArchivedReadingListScreen(),
                  ),
                  GoRoute(
                    path: routeWebClipReaderPath,
                    name: routeWebClipReaderName,
                    parentNavigatorKey: _rootNavigatorKey,
                    builder: (context, state) => WebClipReaderScreen(
                      id: _idParam(state, 'id'),
                      initialTitle: state.extra! as String,
                    ),
                  ),
                  GoRoute(
                    path: routeReadingListFeedEntryReaderPath,
                    name: routeReadingListFeedEntryReaderName,
                    parentNavigatorKey: _rootNavigatorKey,
                    builder: (context, state) => FeedEntryReaderScreen(
                      id: _idParam(state, 'feedEntryId'),
                    ),
                  ),
                ],
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: routeFeedsPath,
                name: routeFeedsName,
                builder: (context, state) => const FeedsScreen(),
                routes: [
                  GoRoute(
                    path: routeFeedSearchPath,
                    name: routeFeedSearchName,
                    builder: (context, state) => const FeedSearchScreen(),
                  ),
                  // The candidate comes from the search result as `extra`, so
                  // neither screen fetches anything of its own.
                  GoRoute(
                    path: routeFeedSubscriptionPath,
                    name: routeFeedSubscriptionName,
                    builder: (context, state) => FeedSubscriptionScreen(
                      candidate: state.extra! as FeedCandidate,
                    ),
                    routes: [
                      GoRoute(
                        path: routeFeedSubscriptionAttributesPath,
                        name: routeFeedSubscriptionAttributesName,
                        builder: (context, state) => AttributePickerScreen(
                          candidate: state.extra! as FeedCandidate,
                          groupId: _idParam(state, 'groupId'),
                        ),
                      ),
                    ],
                  ),
                  GoRoute(
                    path: routeFeedDetailPath,
                    name: routeFeedDetailName,
                    builder: (context, state) =>
                        FeedDetailScreen(id: _idParam(state, 'id')),
                    routes: [
                      GoRoute(
                        path: routeFeedEntryReaderPath,
                        name: routeFeedEntryReaderName,
                        parentNavigatorKey: _rootNavigatorKey,
                        builder: (context, state) => FeedEntryReaderScreen(
                          id: _idParam(state, 'feedEntryId'),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ],
          ),
        ],
      ),
    ],
  );
}

int _idParam(GoRouterState state, String name) =>
    int.tryParse(state.pathParameters[name] ?? '') ?? -1;

/// Sends signed-out users to Sign-in (unless already on a route they may stay
/// on), signed-in users away from Splash/Sign-in/Sign-up/Verify-email to
/// Today, and holds signed-out-or-loading users on Splash while the token read
/// is in flight. A read failure is treated the same as signed-out.
///
/// Verify-email additionally needs [attempt]: without one there is no ticket
/// to verify against, which happens on a deep link or after a hot restart, so
/// the user is sent back to the form that starts an attempt.
String? _authRedirect(
  AsyncValue<String?> sessionAsync,
  GoRouterState state,
  PendingSignUpAttempt? attempt,
) {
  final onAuthRoute = _authRoutePaths.contains(state.matchedLocation);
  return switch (sessionAsync) {
    AsyncData(value: final token?) when token.isNotEmpty =>
      onAuthRoute ? routeTodayPath : null,
    AsyncData() || AsyncError() => switch (state.matchedLocation) {
      routeVerifyEmailPath when attempt == null => routeSignUpPath,
      final path when _signedOutRoutePaths.contains(path) => null,
      _ => routeSignInPath,
    },
    _ => state.matchedLocation == routeSplashPath ? null : routeSplashPath,
  };
}

class const _ScaffoldWithNavBar({
  required final StatefulNavigationShell navigationShell,
}) extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: navigationShell,
      bottomNavigationBar: NavigationBar(
        selectedIndex: navigationShell.currentIndex,
        onDestinationSelected: navigationShell.goBranch,
        destinations: const [
          NavigationDestination(
            key: AppDebugKey.todayNavDestination,
            icon: Icon(Icons.article_outlined),
            selectedIcon: Icon(Icons.article),
            label: 'Today',
          ),
          NavigationDestination(
            key: AppDebugKey.readingListNavDestination,
            icon: Icon(Icons.bookmark_outline),
            selectedIcon: Icon(Icons.bookmark),
            label: 'Reading list',
          ),
          NavigationDestination(
            key: AppDebugKey.feedsNavDestination,
            icon: Icon(Icons.rss_feed),
            label: 'Feeds',
          ),
        ],
      ),
    );
  }
}
