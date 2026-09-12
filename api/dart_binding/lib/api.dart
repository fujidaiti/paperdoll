//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

library openapi.api;

import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:collection/collection.dart';
import 'package:http/http.dart';

part 'api_helper.dart';
part 'model/auth_token.dart';
part 'model/error.dart';
part 'model/feed.dart';
part 'model/feed_candidate.dart';
part 'model/feed_entry.dart';
part 'model/get_feed_timeline200_response.dart';
part 'model/get_feeds200_response.dart';
part 'model/get_reading_list200_response.dart';
part 'model/get_todays_newspaper200_response.dart';
part 'model/get_web_clip200_response.dart';
part 'model/read_later.dart';
part 'model/reading_list_item.dart';
part 'model/resend_sign_up_verification_request.dart';
part 'model/save_to_reading_list_request.dart';
part 'model/save_to_reading_list_request_one_of.dart';
part 'model/save_to_reading_list_request_one_of1.dart';
part 'model/save_to_reading_list_request_one_of2.dart';
part 'model/search_feeds200_response.dart';
part 'model/set_reading_list_item_archived_status_request.dart';
part 'model/sign_in_request.dart';
part 'model/sign_up_request.dart';
part 'model/sign_up_ticket.dart';
part 'model/story.dart';
part 'model/subscribe_to_feed_request.dart';
part 'model/verify_sign_up_email_request.dart';

const _delimiters = {'csv': ',', 'ssv': ' ', 'tsv': '\t', 'pipes': '|'};
const _dateEpochMarker = 'epoch';
const _deepEquality = DeepCollectionEquality();
final _regList = RegExp(r'^List<(.*)>$');
final _regSet = RegExp(r'^Set<(.*)>$');
final _regMap = RegExp(r'^Map<String,(.*)>$');

bool _isEpochMarker(String? pattern) =>
    pattern == _dateEpochMarker || pattern == '/$_dateEpochMarker/';
