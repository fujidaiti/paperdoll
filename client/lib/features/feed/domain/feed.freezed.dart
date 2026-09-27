// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint, type=warning, deprecated_member_use, deprecated_member_use_from_same_package
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'feed.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$Feed {

 int get id; String get url; String get title; bool get subscribed; String? get siteUrl; String? get iconUrl; String? get description;
/// Create a copy of Feed
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$FeedCopyWith<Feed> get copyWith => _$FeedCopyWithImpl<Feed>(this as Feed, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is Feed&&(identical(other.id, id) || other.id == id)&&(identical(other.url, url) || other.url == url)&&(identical(other.title, title) || other.title == title)&&(identical(other.subscribed, subscribed) || other.subscribed == subscribed)&&(identical(other.siteUrl, siteUrl) || other.siteUrl == siteUrl)&&(identical(other.iconUrl, iconUrl) || other.iconUrl == iconUrl)&&(identical(other.description, description) || other.description == description));
}


@override
int get hashCode => Object.hash(runtimeType,id,url,title,subscribed,siteUrl,iconUrl,description);

@override
String toString() {
  return 'Feed(id: $id, url: $url, title: $title, subscribed: $subscribed, siteUrl: $siteUrl, iconUrl: $iconUrl, description: $description)';
}


}

/// @nodoc
abstract mixin class $FeedCopyWith<$Res>  {
  factory $FeedCopyWith(Feed value, $Res Function(Feed) _then) = _$FeedCopyWithImpl;
@useResult
$Res call({
 int id, String url, String title, bool subscribed, String? siteUrl, String? iconUrl, String? description
});




}
/// @nodoc
class _$FeedCopyWithImpl<$Res>
    implements $FeedCopyWith<$Res> {
  _$FeedCopyWithImpl(this._self, this._then);

  final Feed _self;
  final $Res Function(Feed) _then;

/// Create a copy of Feed
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? url = null,Object? title = null,Object? subscribed = null,Object? siteUrl = freezed,Object? iconUrl = freezed,Object? description = freezed,}) {
  return _then(Feed(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as int,url: null == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String,title: null == title ? _self.title : title // ignore: cast_nullable_to_non_nullable
as String,subscribed: null == subscribed ? _self.subscribed : subscribed // ignore: cast_nullable_to_non_nullable
as bool,siteUrl: freezed == siteUrl ? _self.siteUrl : siteUrl // ignore: cast_nullable_to_non_nullable
as String?,iconUrl: freezed == iconUrl ? _self.iconUrl : iconUrl // ignore: cast_nullable_to_non_nullable
as String?,description: freezed == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}

}


/// Adds pattern-matching-related methods to [Feed].
extension FeedPatterns on Feed {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _Feed value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _Feed() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _Feed value)  $default,){
final _that = this;
switch (_that) {
case _Feed():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _Feed value)?  $default,){
final _that = this;
switch (_that) {
case _Feed() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( int id,  String url,  String title,  bool subscribed,  String? siteUrl,  String? iconUrl,  String? description)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _Feed() when $default != null:
return $default(_that.id,_that.url,_that.title,_that.subscribed,_that.siteUrl,_that.iconUrl,_that.description);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( int id,  String url,  String title,  bool subscribed,  String? siteUrl,  String? iconUrl,  String? description)  $default,) {final _that = this;
switch (_that) {
case _Feed():
return $default(_that.id,_that.url,_that.title,_that.subscribed,_that.siteUrl,_that.iconUrl,_that.description);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( int id,  String url,  String title,  bool subscribed,  String? siteUrl,  String? iconUrl,  String? description)?  $default,) {final _that = this;
switch (_that) {
case _Feed() when $default != null:
return $default(_that.id,_that.url,_that.title,_that.subscribed,_that.siteUrl,_that.iconUrl,_that.description);case _:
  return null;

}
}

}

/// @nodoc


class _Feed implements Feed {
  const _Feed({required this.id, required this.url, required this.title, required this.subscribed, this.siteUrl, this.iconUrl, this.description});
  

@override final  int id;
@override final  String url;
@override final  String title;
@override final  bool subscribed;
@override final  String? siteUrl;
@override final  String? iconUrl;
@override final  String? description;

/// Create a copy of Feed
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$FeedCopyWith<_Feed> get copyWith => __$FeedCopyWithImpl<_Feed>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _Feed&&(identical(other.id, id) || other.id == id)&&(identical(other.url, url) || other.url == url)&&(identical(other.title, title) || other.title == title)&&(identical(other.subscribed, subscribed) || other.subscribed == subscribed)&&(identical(other.siteUrl, siteUrl) || other.siteUrl == siteUrl)&&(identical(other.iconUrl, iconUrl) || other.iconUrl == iconUrl)&&(identical(other.description, description) || other.description == description));
}


@override
int get hashCode => Object.hash(runtimeType,id,url,title,subscribed,siteUrl,iconUrl,description);

@override
String toString() {
  return 'Feed(id: $id, url: $url, title: $title, subscribed: $subscribed, siteUrl: $siteUrl, iconUrl: $iconUrl, description: $description)';
}


}

/// @nodoc
abstract mixin class _$FeedCopyWith<$Res> implements $FeedCopyWith<$Res> {
  factory _$FeedCopyWith(_Feed value, $Res Function(_Feed) _then) = __$FeedCopyWithImpl;
@override @useResult
$Res call({
 int id, String url, String title, bool subscribed, String? siteUrl, String? iconUrl, String? description
});




}
/// @nodoc
class __$FeedCopyWithImpl<$Res>
    implements _$FeedCopyWith<$Res> {
  __$FeedCopyWithImpl(this._self, this._then);

  final _Feed _self;
  final $Res Function(_Feed) _then;

/// Create a copy of Feed
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? url = null,Object? title = null,Object? subscribed = null,Object? siteUrl = freezed,Object? iconUrl = freezed,Object? description = freezed,}) {
  return _then(_Feed(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as int,url: null == url ? _self.url : url // ignore: cast_nullable_to_non_nullable
as String,title: null == title ? _self.title : title // ignore: cast_nullable_to_non_nullable
as String,subscribed: null == subscribed ? _self.subscribed : subscribed // ignore: cast_nullable_to_non_nullable
as bool,siteUrl: freezed == siteUrl ? _self.siteUrl : siteUrl // ignore: cast_nullable_to_non_nullable
as String?,iconUrl: freezed == iconUrl ? _self.iconUrl : iconUrl // ignore: cast_nullable_to_non_nullable
as String?,description: freezed == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}


}

// dart format on
