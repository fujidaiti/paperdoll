// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint, type=warning, deprecated_member_use, deprecated_member_use_from_same_package
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'post_selection.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$PostSelection {

 int get groupId; String? get link; String? get title; String? get description; String? get image; String? get timestamp;/// Whether the user has answered any attribute other than the link, even
/// with "None of them". Until then the group preview shows a guess.
 bool get answered;
/// Create a copy of PostSelection
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$PostSelectionCopyWith<PostSelection> get copyWith => _$PostSelectionCopyWithImpl<PostSelection>(this as PostSelection, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is PostSelection&&(identical(other.groupId, groupId) || other.groupId == groupId)&&(identical(other.link, link) || other.link == link)&&(identical(other.title, title) || other.title == title)&&(identical(other.description, description) || other.description == description)&&(identical(other.image, image) || other.image == image)&&(identical(other.timestamp, timestamp) || other.timestamp == timestamp)&&(identical(other.answered, answered) || other.answered == answered));
}


@override
int get hashCode => Object.hash(runtimeType,groupId,link,title,description,image,timestamp,answered);

@override
String toString() {
  return 'PostSelection(groupId: $groupId, link: $link, title: $title, description: $description, image: $image, timestamp: $timestamp, answered: $answered)';
}


}

/// @nodoc
abstract mixin class $PostSelectionCopyWith<$Res>  {
  factory $PostSelectionCopyWith(PostSelection value, $Res Function(PostSelection) _then) = _$PostSelectionCopyWithImpl;
@useResult
$Res call({
 int groupId, String? link, String? title, String? description, String? image, String? timestamp, bool answered
});




}
/// @nodoc
class _$PostSelectionCopyWithImpl<$Res>
    implements $PostSelectionCopyWith<$Res> {
  _$PostSelectionCopyWithImpl(this._self, this._then);

  final PostSelection _self;
  final $Res Function(PostSelection) _then;

/// Create a copy of PostSelection
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? groupId = null,Object? link = freezed,Object? title = freezed,Object? description = freezed,Object? image = freezed,Object? timestamp = freezed,Object? answered = null,}) {
  return _then(PostSelection(
groupId: null == groupId ? _self.groupId : groupId // ignore: cast_nullable_to_non_nullable
as int,link: freezed == link ? _self.link : link // ignore: cast_nullable_to_non_nullable
as String?,title: freezed == title ? _self.title : title // ignore: cast_nullable_to_non_nullable
as String?,description: freezed == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String?,image: freezed == image ? _self.image : image // ignore: cast_nullable_to_non_nullable
as String?,timestamp: freezed == timestamp ? _self.timestamp : timestamp // ignore: cast_nullable_to_non_nullable
as String?,answered: null == answered ? _self.answered : answered // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}

}


/// Adds pattern-matching-related methods to [PostSelection].
extension PostSelectionPatterns on PostSelection {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _PostSelection value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _PostSelection() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _PostSelection value)  $default,){
final _that = this;
switch (_that) {
case _PostSelection():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _PostSelection value)?  $default,){
final _that = this;
switch (_that) {
case _PostSelection() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( int groupId,  String? link,  String? title,  String? description,  String? image,  String? timestamp,  bool answered)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _PostSelection() when $default != null:
return $default(_that.groupId,_that.link,_that.title,_that.description,_that.image,_that.timestamp,_that.answered);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( int groupId,  String? link,  String? title,  String? description,  String? image,  String? timestamp,  bool answered)  $default,) {final _that = this;
switch (_that) {
case _PostSelection():
return $default(_that.groupId,_that.link,_that.title,_that.description,_that.image,_that.timestamp,_that.answered);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( int groupId,  String? link,  String? title,  String? description,  String? image,  String? timestamp,  bool answered)?  $default,) {final _that = this;
switch (_that) {
case _PostSelection() when $default != null:
return $default(_that.groupId,_that.link,_that.title,_that.description,_that.image,_that.timestamp,_that.answered);case _:
  return null;

}
}

}

/// @nodoc


class _PostSelection implements PostSelection {
  const _PostSelection({required this.groupId, this.link, this.title, this.description, this.image, this.timestamp, this.answered = false});
  

@override final  int groupId;
@override final  String? link;
@override final  String? title;
@override final  String? description;
@override final  String? image;
@override final  String? timestamp;
/// Whether the user has answered any attribute other than the link, even
/// with "None of them". Until then the group preview shows a guess.
@override@JsonKey() final  bool answered;

/// Create a copy of PostSelection
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$PostSelectionCopyWith<_PostSelection> get copyWith => __$PostSelectionCopyWithImpl<_PostSelection>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _PostSelection&&(identical(other.groupId, groupId) || other.groupId == groupId)&&(identical(other.link, link) || other.link == link)&&(identical(other.title, title) || other.title == title)&&(identical(other.description, description) || other.description == description)&&(identical(other.image, image) || other.image == image)&&(identical(other.timestamp, timestamp) || other.timestamp == timestamp)&&(identical(other.answered, answered) || other.answered == answered));
}


@override
int get hashCode => Object.hash(runtimeType,groupId,link,title,description,image,timestamp,answered);

@override
String toString() {
  return 'PostSelection(groupId: $groupId, link: $link, title: $title, description: $description, image: $image, timestamp: $timestamp, answered: $answered)';
}


}

/// @nodoc
abstract mixin class _$PostSelectionCopyWith<$Res> implements $PostSelectionCopyWith<$Res> {
  factory _$PostSelectionCopyWith(_PostSelection value, $Res Function(_PostSelection) _then) = __$PostSelectionCopyWithImpl;
@override @useResult
$Res call({
 int groupId, String? link, String? title, String? description, String? image, String? timestamp, bool answered
});




}
/// @nodoc
class __$PostSelectionCopyWithImpl<$Res>
    implements _$PostSelectionCopyWith<$Res> {
  __$PostSelectionCopyWithImpl(this._self, this._then);

  final _PostSelection _self;
  final $Res Function(_PostSelection) _then;

/// Create a copy of PostSelection
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? groupId = null,Object? link = freezed,Object? title = freezed,Object? description = freezed,Object? image = freezed,Object? timestamp = freezed,Object? answered = null,}) {
  return _then(_PostSelection(
groupId: null == groupId ? _self.groupId : groupId // ignore: cast_nullable_to_non_nullable
as int,link: freezed == link ? _self.link : link // ignore: cast_nullable_to_non_nullable
as String?,title: freezed == title ? _self.title : title // ignore: cast_nullable_to_non_nullable
as String?,description: freezed == description ? _self.description : description // ignore: cast_nullable_to_non_nullable
as String?,image: freezed == image ? _self.image : image // ignore: cast_nullable_to_non_nullable
as String?,timestamp: freezed == timestamp ? _self.timestamp : timestamp // ignore: cast_nullable_to_non_nullable
as String?,answered: null == answered ? _self.answered : answered // ignore: cast_nullable_to_non_nullable
as bool,
  ));
}


}

// dart format on
