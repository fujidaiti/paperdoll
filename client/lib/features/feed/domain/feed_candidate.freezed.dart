// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint, type=warning, deprecated_member_use, deprecated_member_use_from_same_package
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'feed_candidate.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$FeedCandidate {

 String get url; String get title; String? get siteUrl; String? get iconUrl; String? get description; List<PostGroup> get postGroups;
/// Create a copy of FeedCandidate
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$FeedCandidateCopyWith<FeedCandidate> get copyWith => _$FeedCandidateCopyWithImpl<FeedCandidate>(this as FeedCandidate, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is FeedCandidate&&(identical(other.url, url) || other.url == url)&&(identical(other.title, title) || other.title == title)&&(identical(other.siteUrl, siteUrl) || other.siteUrl == siteUrl)&&(identical(other.iconUrl, iconUrl) || other.iconUrl == iconUrl)&&(identical(other.description, description) || other.description == description)&&const DeepCollectionEquality().equals(other.postGroups, postGroups));
}


@override
int get hashCode => Object.hash(runtimeType,url,title,siteUrl,iconUrl,description,const DeepCollectionEquality().hash(postGroups));

@override
String toString() {
  return 'FeedCandidate(url: $url, title: $title, siteUrl: $siteUrl, iconUrl: $iconUrl, description: $description, postGroups: $postGroups)';
}


}

/// @nodoc
abstract mixin class $FeedCandidateCopyWith<$Res>  {
  factory $FeedCandidateCopyWith(FeedCandidate value, $Res Function(FeedCandidate) _then) = _$FeedCandidateCopyWithImpl;
@useResult
$Res call({
 String url, String title, String? siteUrl, String? iconUrl, String? description, List<PostGroup> postGroups
});




}
/// @nodoc
class _$FeedCandidateCopyWithImpl<$Res>
    implements $FeedCandidateCopyWith<$Res> {
  _$FeedCandidateCopyWithImpl(this._self, this._then);

  final FeedCandidate _self;
  final $Res Function(FeedCandidate) _then;

/// Create a copy of FeedCandidate
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? url = null,Object? title = null,Object? siteUrl = freezed,Object? iconUrl = freezed,Object? description = freezed,Object? postGroups = null,}) {
  return _then(FeedCandidate(
url: null == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String,title: null == title ? _self.title : title // ignore: cast_nullable_to_non_nullable
as String,siteUrl: freezed == siteUrl ? _self.siteUrl : siteUrl // ignore: cast_nullable_to_non_nullable
as String?,iconUrl: freezed == iconUrl ? _self.iconUrl : iconUrl // ignore: cast_nullable_to_non_nullable
as String?,description: freezed == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String?,postGroups: null == postGroups ? _self.postGroups : postGroups // ignore: cast_nullable_to_non_nullable
as List<PostGroup>,
  ));
}

}


/// Adds pattern-matching-related methods to [FeedCandidate].
extension FeedCandidatePatterns on FeedCandidate {
/// A variant of `map` that fallback to returning `orElse`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _FeedCandidate value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _FeedCandidate() when $default != null:
return $default(_that);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// Callbacks receives the raw object, upcasted.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case final Subclass2 value:
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _FeedCandidate value)  $default,){
final _that = this;
switch (_that) {
case _FeedCandidate():
return $default(_that);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `map` that fallback to returning `null`.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case final Subclass value:
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _FeedCandidate value)?  $default,){
final _that = this;
switch (_that) {
case _FeedCandidate() when $default != null:
return $default(_that);case _:
  return null;

}
}
/// A variant of `when` that fallback to an `orElse` callback.
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return orElse();
/// }
/// ```

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String url,  String title,  String? siteUrl,  String? iconUrl,  String? description,  List<PostGroup> postGroups)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _FeedCandidate() when $default != null:
return $default(_that.url,_that.title,_that.siteUrl,_that.iconUrl,_that.description,_that.postGroups);case _:
  return orElse();

}
}
/// A `switch`-like method, using callbacks.
///
/// As opposed to `map`, this offers destructuring.
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case Subclass2(:final field2):
///     return ...;
/// }
/// ```

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String url,  String title,  String? siteUrl,  String? iconUrl,  String? description,  List<PostGroup> postGroups)  $default,) {final _that = this;
switch (_that) {
case _FeedCandidate():
return $default(_that.url,_that.title,_that.siteUrl,_that.iconUrl,_that.description,_that.postGroups);case _:
  throw StateError('Unexpected subclass');

}
}
/// A variant of `when` that fallback to returning `null`
///
/// It is equivalent to doing:
/// ```dart
/// switch (sealedClass) {
///   case Subclass(:final field):
///     return ...;
///   case _:
///     return null;
/// }
/// ```

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String url,  String title,  String? siteUrl,  String? iconUrl,  String? description,  List<PostGroup> postGroups)?  $default,) {final _that = this;
switch (_that) {
case _FeedCandidate() when $default != null:
return $default(_that.url,_that.title,_that.siteUrl,_that.iconUrl,_that.description,_that.postGroups);case _:
  return null;

}
}

}

/// @nodoc


class _FeedCandidate implements FeedCandidate {
  const _FeedCandidate({required this.url, required this.title, this.siteUrl, this.iconUrl, this.description,  List<PostGroup> postGroups = const []}): _postGroups = postGroups;
  

@override final  String url;
@override final  String title;
@override final  String? siteUrl;
@override final  String? iconUrl;
@override final  String? description;
 final  List<PostGroup> _postGroups;
@override@JsonKey() List<PostGroup> get postGroups {
  if (_postGroups is EqualUnmodifiableListView) return _postGroups;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_postGroups);
}


/// Create a copy of FeedCandidate
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$FeedCandidateCopyWith<_FeedCandidate> get copyWith => __$FeedCandidateCopyWithImpl<_FeedCandidate>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _FeedCandidate&&(identical(other.url, url) || other.url == url)&&(identical(other.title, title) || other.title == title)&&(identical(other.siteUrl, siteUrl) || other.siteUrl == siteUrl)&&(identical(other.iconUrl, iconUrl) || other.iconUrl == iconUrl)&&(identical(other.description, description) || other.description == description)&&const DeepCollectionEquality().equals(other._postGroups, _postGroups));
}


@override
int get hashCode => Object.hash(runtimeType,url,title,siteUrl,iconUrl,description,const DeepCollectionEquality().hash(_postGroups));

@override
String toString() {
  return 'FeedCandidate(url: $url, title: $title, siteUrl: $siteUrl, iconUrl: $iconUrl, description: $description, postGroups: $postGroups)';
}


}

/// @nodoc
abstract mixin class _$FeedCandidateCopyWith<$Res> implements $FeedCandidateCopyWith<$Res> {
  factory _$FeedCandidateCopyWith(_FeedCandidate value, $Res Function(_FeedCandidate) _then) = __$FeedCandidateCopyWithImpl;
@override @useResult
$Res call({
 String url, String title, String? siteUrl, String? iconUrl, String? description, List<PostGroup> postGroups
});




}
/// @nodoc
class __$FeedCandidateCopyWithImpl<$Res>
    implements _$FeedCandidateCopyWith<$Res> {
  __$FeedCandidateCopyWithImpl(this._self, this._then);

  final _FeedCandidate _self;
  final $Res Function(_FeedCandidate) _then;

/// Create a copy of FeedCandidate
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? url = null,Object? title = null,Object? siteUrl = freezed,Object? iconUrl = freezed,Object? description = freezed,Object? postGroups = null,}) {
  return _then(_FeedCandidate(
url: null == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String,title: null == title ? _self.title : title // ignore: cast_nullable_to_non_nullable
as String,siteUrl: freezed == siteUrl ? _self.siteUrl : siteUrl // ignore: cast_nullable_to_non_nullable
as String?,iconUrl: freezed == iconUrl ? _self.iconUrl : iconUrl // ignore: cast_nullable_to_non_nullable
as String?,description: freezed == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String?,postGroups: null == postGroups ? _self._postGroups : postGroups // ignore: cast_nullable_to_non_nullable
as List<PostGroup>,
  ));
}


}

// dart format on
