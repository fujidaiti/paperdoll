import 'package:flutter/widgets.dart';

// Intentional namespace class — Dart has no dedicated namespace syntax.
// ignore: avoid_classes_with_only_static_members
abstract final class AppDebugKey {
  static const addFeedButton = Key('addFeedButton');
  static const archiveSuccessSnackBar = Key('archiveSuccessSnackBar');
  static const archivedButton = Key('archivedButton');
  static const archivedReadingListScreen = Key('archivedReadingListScreen');
  static const unarchiveSuccessSnackBar = Key('unarchiveSuccessSnackBar');
  static const feedDetailScreen = Key('feedDetailScreen');
  static const feedDetailMenuButton = Key('feedDetailMenuButton');
  static const subscribeMenuItem = Key('subscribeMenuItem');
  static const unsubscribeMenuItem = Key('unsubscribeMenuItem');
  static const feedEntryReaderArchiveButton = Key(
    'feedEntryReaderArchiveButton',
  );
  static const feedEntryReaderArchivedBanner = Key(
    'feedEntryReaderArchivedBanner',
  );
  static const feedEntryReaderBookmarkButton = Key(
    'feedEntryReaderBookmarkButton',
  );
  static const feedEntryReaderOpenOriginalButton = Key(
    'feedEntryReaderOpenOriginalButton',
  );
  static const feedEntryReaderScreen = Key('feedEntryReaderScreen');
  static const feedSearchButton = Key('feedSearchButton');
  static const feedSearchScreen = Key('feedSearchScreen');
  static const feedSearchTextField = Key('feedSearchTextField');
  static const feedsNavDestination = Key('feedsNavDestination');
  static const feedsScreen = Key('feedsScreen');
  static const readingListNavDestination = Key('readingListNavDestination');
  static const readingListScreen = Key('readingListScreen');
  static const webClipReaderScreen = Key('webClipReaderScreen');
  static const webClipReaderArchiveButton = Key('webClipReaderArchiveButton');
  static const webClipReaderArchivedBanner = Key('webClipReaderArchivedBanner');
  static const webClipReaderBookmarkButton = Key('webClipReaderBookmarkButton');
  static const webClipReaderOpenOriginalButton = Key(
    'webClipReaderOpenOriginalButton',
  );
  static const removeFromReadingListSuccessSnackBar = Key(
    'removeFromReadingListSuccessSnackBar',
  );
  static const saveToReadingListSuccessSnackBar = Key(
    'saveToReadingListSuccessSnackBar',
  );
  static const settingsButton = Key('settingsButton');
  static const settingsScreen = Key('settingsScreen');
  static const signOutButton = Key('signOutButton');
  static const signInScreen = Key('signInScreen');
  static const signInEmailField = Key('signInEmailField');
  static const signInPasswordField = Key('signInPasswordField');
  static const signInSubmitButton = Key('signInSubmitButton');
  static const signInGoToSignUpButton = Key('signInGoToSignUpButton');
  static const signUpScreen = Key('signUpScreen');
  static const signUpEmailField = Key('signUpEmailField');
  static const signUpPasswordField = Key('signUpPasswordField');
  static const signUpSubmitButton = Key('signUpSubmitButton');
  static const signUpGoToSignInButton = Key('signUpGoToSignInButton');
  static const subscribeSuccessSnackBar = Key('subscribeSuccessSnackBar');
  static const unsubscribeSuccessSnackBar = Key('unsubscribeSuccessSnackBar');
  static const verifyEmailScreen = Key('verifyEmailScreen');
  static const verifyEmailCodeField = Key('verifyEmailCodeField');
  static const verifyEmailSubmitButton = Key('verifyEmailSubmitButton');
  static const verifyEmailResendButton = Key('verifyEmailResendButton');
  static const verifyEmailCodeSentSnackBar = Key('verifyEmailCodeSentSnackBar');
  static const verifyEmailStartOverButton = Key('verifyEmailStartOverButton');
  static const verifyEmailGoToSignInButton = Key('verifyEmailGoToSignInButton');
  static const todayNavDestination = Key('todayNavDestination');
  static const todayScreen = Key('todayScreen');

  static Key feedCandidateTile(String title) => Key('feedCandidate:$title');
  static Key feedEntryRow(String title) => Key('feedEntry:$title');
  static Key feedRow(String title) => Key('feed:$title');
  static Key readingListRow(String title) => Key('readingList:$title');
  static Key readerTitle(String title) => Key('readerTitle:$title');
  static Key storyCard(String title) => Key('story:$title');
}
