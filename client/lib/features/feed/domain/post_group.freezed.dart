// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint, type=warning, deprecated_member_use, deprecated_member_use_from_same_package
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'post_group.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
T _$identity<T>(T value) => value;
/// @nodoc
mixin _$PostGroup {

/// Unique inside one search response only. Never sent back.
 int get id;/// CSS selector matching the items. Sent back as `PostSelectors.root`.
 String get selector;/// How many items the group holds in the page.
 int get count;/// How many items each [AttributeCandidate.values] describes. The sample
/// always covers every candidate of the group, so a row is never empty in
/// all of the sampled items.
 int get sampled; List<AttributeCandidate> get links; List<AttributeCandidate> get texts; List<AttributeCandidate> get images;
/// Create a copy of PostGroup
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$PostGroupCopyWith<PostGroup> get copyWith => _$PostGroupCopyWithImpl<PostGroup>(this as PostGroup, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is PostGroup&&(identical(other.id, id) || other.id == id)&&(identical(other.selector, selector) || other.selector == selector)&&(identical(other.count, count) || other.count == count)&&(identical(other.sampled, sampled) || other.sampled == sampled)&&const DeepCollectionEquality().equals(other.links, links)&&const DeepCollectionEquality().equals(other.texts, texts)&&const DeepCollectionEquality().equals(other.images, images));
}


@override
int get hashCode => Object.hash(runtimeType,id,selector,count,sampled,const DeepCollectionEquality().hash(links),const DeepCollectionEquality().hash(texts),const DeepCollectionEquality().hash(images));

@override
String toString() {
  return 'PostGroup(id: $id, selector: $selector, count: $count, sampled: $sampled, links: $links, texts: $texts, images: $images)';
}


}

/// @nodoc
abstract mixin class $PostGroupCopyWith<$Res>  {
  factory $PostGroupCopyWith(PostGroup value, $Res Function(PostGroup) _then) = _$PostGroupCopyWithImpl;
@useResult
$Res call({
 int id, String selector, int count, int sampled, List<AttributeCandidate> links, List<AttributeCandidate> texts, List<AttributeCandidate> images
});




}
/// @nodoc
class _$PostGroupCopyWithImpl<$Res>
    implements $PostGroupCopyWith<$Res> {
  _$PostGroupCopyWithImpl(this._self, this._then);

  final PostGroup _self;
  final $Res Function(PostGroup) _then;

/// Create a copy of PostGroup
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? id = null,Object? selector = null,Object? count = null,Object? sampled = null,Object? links = null,Object? texts = null,Object? images = null,}) {
  return _then(PostGroup(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as int,selector: null == selector ? _self.selector : selector // ignore: cast_nullable_to_non_nullable
as String,count: null == count ? _self.count : count // ignore: cast_nullable_to_non_nullable
as int,sampled: null == sampled ? _self.sampled : sampled // ignore: cast_nullable_to_non_nullable
as int,links: null == links ? _self.links : links // ignore: cast_nullable_to_non_nullable
as List<AttributeCandidate>,texts: null == texts ? _self.texts : texts // ignore: cast_nullable_to_non_nullable
as List<AttributeCandidate>,images: null == images ? _self.images : images // ignore: cast_nullable_to_non_nullable
as List<AttributeCandidate>,
  ));
}

}


/// Adds pattern-matching-related methods to [PostGroup].
extension PostGroupPatterns on PostGroup {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _PostGroup value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _PostGroup() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _PostGroup value)  $default,){
final _that = this;
switch (_that) {
case _PostGroup():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _PostGroup value)?  $default,){
final _that = this;
switch (_that) {
case _PostGroup() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( int id,  String selector,  int count,  int sampled,  List<AttributeCandidate> links,  List<AttributeCandidate> texts,  List<AttributeCandidate> images)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _PostGroup() when $default != null:
return $default(_that.id,_that.selector,_that.count,_that.sampled,_that.links,_that.texts,_that.images);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( int id,  String selector,  int count,  int sampled,  List<AttributeCandidate> links,  List<AttributeCandidate> texts,  List<AttributeCandidate> images)  $default,) {final _that = this;
switch (_that) {
case _PostGroup():
return $default(_that.id,_that.selector,_that.count,_that.sampled,_that.links,_that.texts,_that.images);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( int id,  String selector,  int count,  int sampled,  List<AttributeCandidate> links,  List<AttributeCandidate> texts,  List<AttributeCandidate> images)?  $default,) {final _that = this;
switch (_that) {
case _PostGroup() when $default != null:
return $default(_that.id,_that.selector,_that.count,_that.sampled,_that.links,_that.texts,_that.images);case _:
  return null;

}
}

}

/// @nodoc


class _PostGroup implements PostGroup {
  const _PostGroup({required this.id, required this.selector, required this.count, required this.sampled, required  List<AttributeCandidate> links, required  List<AttributeCandidate> texts, required  List<AttributeCandidate> images}): _links = links,_texts = texts,_images = images;
  

/// Unique inside one search response only. Never sent back.
@override final  int id;
/// CSS selector matching the items. Sent back as `PostSelectors.root`.
@override final  String selector;
/// How many items the group holds in the page.
@override final  int count;
/// How many items each [AttributeCandidate.values] describes. The sample
/// always covers every candidate of the group, so a row is never empty in
/// all of the sampled items.
@override final  int sampled;
 final  List<AttributeCandidate> _links;
@override List<AttributeCandidate> get links {
  if (_links is EqualUnmodifiableListView) return _links;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_links);
}

 final  List<AttributeCandidate> _texts;
@override List<AttributeCandidate> get texts {
  if (_texts is EqualUnmodifiableListView) return _texts;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_texts);
}

 final  List<AttributeCandidate> _images;
@override List<AttributeCandidate> get images {
  if (_images is EqualUnmodifiableListView) return _images;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_images);
}


/// Create a copy of PostGroup
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$PostGroupCopyWith<_PostGroup> get copyWith => __$PostGroupCopyWithImpl<_PostGroup>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _PostGroup&&(identical(other.id, id) || other.id == id)&&(identical(other.selector, selector) || other.selector == selector)&&(identical(other.count, count) || other.count == count)&&(identical(other.sampled, sampled) || other.sampled == sampled)&&const DeepCollectionEquality().equals(other._links, _links)&&const DeepCollectionEquality().equals(other._texts, _texts)&&const DeepCollectionEquality().equals(other._images, _images));
}


@override
int get hashCode => Object.hash(runtimeType,id,selector,count,sampled,const DeepCollectionEquality().hash(_links),const DeepCollectionEquality().hash(_texts),const DeepCollectionEquality().hash(_images));

@override
String toString() {
  return 'PostGroup(id: $id, selector: $selector, count: $count, sampled: $sampled, links: $links, texts: $texts, images: $images)';
}


}

/// @nodoc
abstract mixin class _$PostGroupCopyWith<$Res> implements $PostGroupCopyWith<$Res> {
  factory _$PostGroupCopyWith(_PostGroup value, $Res Function(_PostGroup) _then) = __$PostGroupCopyWithImpl;
@override @useResult
$Res call({
 int id, String selector, int count, int sampled, List<AttributeCandidate> links, List<AttributeCandidate> texts, List<AttributeCandidate> images
});




}
/// @nodoc
class __$PostGroupCopyWithImpl<$Res>
    implements _$PostGroupCopyWith<$Res> {
  __$PostGroupCopyWithImpl(this._self, this._then);

  final _PostGroup _self;
  final $Res Function(_PostGroup) _then;

/// Create a copy of PostGroup
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? id = null,Object? selector = null,Object? count = null,Object? sampled = null,Object? links = null,Object? texts = null,Object? images = null,}) {
  return _then(_PostGroup(
id: null == id ? _self.id : id // ignore: cast_nullable_to_non_nullable
as int,selector: null == selector ? _self.selector : selector // ignore: cast_nullable_to_non_nullable
as String,count: null == count ? _self.count : count // ignore: cast_nullable_to_non_nullable
as int,sampled: null == sampled ? _self.sampled : sampled // ignore: cast_nullable_to_non_nullable
as int,links: null == links ? _self._links : links // ignore: cast_nullable_to_non_nullable
as List<AttributeCandidate>,texts: null == texts ? _self._texts : texts // ignore: cast_nullable_to_non_nullable
as List<AttributeCandidate>,images: null == images ? _self._images : images // ignore: cast_nullable_to_non_nullable
as List<AttributeCandidate>,
  ));
}


}

/// @nodoc
mixin _$AttributeCandidate {

/// Relative to the item. `:scope` means the item itself.
 String get selector;/// Exactly `PostGroup.sampled` entries, in item order. An entry with a
/// null value means the selector reaches nothing in that item, which
/// happens for a selector only some items of the group carry.
 List<AttributeValue> get values;
/// Create a copy of AttributeCandidate
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$AttributeCandidateCopyWith<AttributeCandidate> get copyWith => _$AttributeCandidateCopyWithImpl<AttributeCandidate>(this as AttributeCandidate, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is AttributeCandidate&&(identical(other.selector, selector) || other.selector == selector)&&const DeepCollectionEquality().equals(other.values, values));
}


@override
int get hashCode => Object.hash(runtimeType,selector,const DeepCollectionEquality().hash(values));

@override
String toString() {
  return 'AttributeCandidate(selector: $selector, values: $values)';
}


}

/// @nodoc
abstract mixin class $AttributeCandidateCopyWith<$Res>  {
  factory $AttributeCandidateCopyWith(AttributeCandidate value, $Res Function(AttributeCandidate) _then) = _$AttributeCandidateCopyWithImpl;
@useResult
$Res call({
 String selector, List<AttributeValue> values
});




}
/// @nodoc
class _$AttributeCandidateCopyWithImpl<$Res>
    implements $AttributeCandidateCopyWith<$Res> {
  _$AttributeCandidateCopyWithImpl(this._self, this._then);

  final AttributeCandidate _self;
  final $Res Function(AttributeCandidate) _then;

/// Create a copy of AttributeCandidate
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? selector = null,Object? values = null,}) {
  return _then(AttributeCandidate(
selector: null == selector ? _self.selector : selector // ignore: cast_nullable_to_non_nullable
as String,values: null == values ? _self.values : values // ignore: cast_nullable_to_non_nullable
as List<AttributeValue>,
  ));
}

}


/// Adds pattern-matching-related methods to [AttributeCandidate].
extension AttributeCandidatePatterns on AttributeCandidate {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _AttributeCandidate value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _AttributeCandidate() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _AttributeCandidate value)  $default,){
final _that = this;
switch (_that) {
case _AttributeCandidate():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _AttributeCandidate value)?  $default,){
final _that = this;
switch (_that) {
case _AttributeCandidate() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String selector,  List<AttributeValue> values)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _AttributeCandidate() when $default != null:
return $default(_that.selector,_that.values);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String selector,  List<AttributeValue> values)  $default,) {final _that = this;
switch (_that) {
case _AttributeCandidate():
return $default(_that.selector,_that.values);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String selector,  List<AttributeValue> values)?  $default,) {final _that = this;
switch (_that) {
case _AttributeCandidate() when $default != null:
return $default(_that.selector,_that.values);case _:
  return null;

}
}

}

/// @nodoc


class _AttributeCandidate implements AttributeCandidate {
  const _AttributeCandidate({required this.selector, required  List<AttributeValue> values}): _values = values;
  

/// Relative to the item. `:scope` means the item itself.
@override final  String selector;
/// Exactly `PostGroup.sampled` entries, in item order. An entry with a
/// null value means the selector reaches nothing in that item, which
/// happens for a selector only some items of the group carry.
 final  List<AttributeValue> _values;
/// Exactly `PostGroup.sampled` entries, in item order. An entry with a
/// null value means the selector reaches nothing in that item, which
/// happens for a selector only some items of the group carry.
@override List<AttributeValue> get values {
  if (_values is EqualUnmodifiableListView) return _values;
  // ignore: implicit_dynamic_type
  return EqualUnmodifiableListView(_values);
}


/// Create a copy of AttributeCandidate
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$AttributeCandidateCopyWith<_AttributeCandidate> get copyWith => __$AttributeCandidateCopyWithImpl<_AttributeCandidate>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _AttributeCandidate&&(identical(other.selector, selector) || other.selector == selector)&&const DeepCollectionEquality().equals(other._values, _values));
}


@override
int get hashCode => Object.hash(runtimeType,selector,const DeepCollectionEquality().hash(_values));

@override
String toString() {
  return 'AttributeCandidate(selector: $selector, values: $values)';
}


}

/// @nodoc
abstract mixin class _$AttributeCandidateCopyWith<$Res> implements $AttributeCandidateCopyWith<$Res> {
  factory _$AttributeCandidateCopyWith(_AttributeCandidate value, $Res Function(_AttributeCandidate) _then) = __$AttributeCandidateCopyWithImpl;
@override @useResult
$Res call({
 String selector, List<AttributeValue> values
});




}
/// @nodoc
class __$AttributeCandidateCopyWithImpl<$Res>
    implements _$AttributeCandidateCopyWith<$Res> {
  __$AttributeCandidateCopyWithImpl(this._self, this._then);

  final _AttributeCandidate _self;
  final $Res Function(_AttributeCandidate) _then;

/// Create a copy of AttributeCandidate
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? selector = null,Object? values = null,}) {
  return _then(_AttributeCandidate(
selector: null == selector ? _self.selector : selector // ignore: cast_nullable_to_non_nullable
as String,values: null == values ? _self._values : values // ignore: cast_nullable_to_non_nullable
as List<AttributeValue>,
  ));
}


}

/// @nodoc
mixin _$AttributeValue {

/// Text, resolved href, or resolved image URL. Null when the selector
/// reaches nothing in that item.
 String? get value;/// Images only.
 String? get alt;
/// Create a copy of AttributeValue
/// with the given fields replaced by the non-null parameter values.
@JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
$AttributeValueCopyWith<AttributeValue> get copyWith => _$AttributeValueCopyWithImpl<AttributeValue>(this as AttributeValue, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is AttributeValue&&(identical(other.value, value) || other.value == value)&&(identical(other.alt, alt) || other.alt == alt));
}


@override
int get hashCode => Object.hash(runtimeType,value,alt);

@override
String toString() {
  return 'AttributeValue(value: $value, alt: $alt)';
}


}

/// @nodoc
abstract mixin class $AttributeValueCopyWith<$Res>  {
  factory $AttributeValueCopyWith(AttributeValue value, $Res Function(AttributeValue) _then) = _$AttributeValueCopyWithImpl;
@useResult
$Res call({
 String? value, String? alt
});




}
/// @nodoc
class _$AttributeValueCopyWithImpl<$Res>
    implements $AttributeValueCopyWith<$Res> {
  _$AttributeValueCopyWithImpl(this._self, this._then);

  final AttributeValue _self;
  final $Res Function(AttributeValue) _then;

/// Create a copy of AttributeValue
/// with the given fields replaced by the non-null parameter values.
@pragma('vm:prefer-inline') @override $Res call({Object? value = freezed,Object? alt = freezed,}) {
  return _then(AttributeValue(
value: freezed == value ? _self.value : value // ignore: cast_nullable_to_non_nullable
as String?,alt: freezed == alt ? _self.alt : alt // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}

}


/// Adds pattern-matching-related methods to [AttributeValue].
extension AttributeValuePatterns on AttributeValue {
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

@optionalTypeArgs TResult maybeMap<TResult extends Object?>(TResult Function( _AttributeValue value)?  $default,{required TResult orElse(),}){
final _that = this;
switch (_that) {
case _AttributeValue() when $default != null:
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

@optionalTypeArgs TResult map<TResult extends Object?>(TResult Function( _AttributeValue value)  $default,){
final _that = this;
switch (_that) {
case _AttributeValue():
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

@optionalTypeArgs TResult? mapOrNull<TResult extends Object?>(TResult? Function( _AttributeValue value)?  $default,){
final _that = this;
switch (_that) {
case _AttributeValue() when $default != null:
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

@optionalTypeArgs TResult maybeWhen<TResult extends Object?>(TResult Function( String? value,  String? alt)?  $default,{required TResult orElse(),}) {final _that = this;
switch (_that) {
case _AttributeValue() when $default != null:
return $default(_that.value,_that.alt);case _:
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

@optionalTypeArgs TResult when<TResult extends Object?>(TResult Function( String? value,  String? alt)  $default,) {final _that = this;
switch (_that) {
case _AttributeValue():
return $default(_that.value,_that.alt);case _:
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

@optionalTypeArgs TResult? whenOrNull<TResult extends Object?>(TResult? Function( String? value,  String? alt)?  $default,) {final _that = this;
switch (_that) {
case _AttributeValue() when $default != null:
return $default(_that.value,_that.alt);case _:
  return null;

}
}

}

/// @nodoc


class _AttributeValue implements AttributeValue {
  const _AttributeValue({this.value, this.alt});
  

/// Text, resolved href, or resolved image URL. Null when the selector
/// reaches nothing in that item.
@override final  String? value;
/// Images only.
@override final  String? alt;

/// Create a copy of AttributeValue
/// with the given fields replaced by the non-null parameter values.
@override @JsonKey(includeFromJson: false, includeToJson: false)
@pragma('vm:prefer-inline')
_$AttributeValueCopyWith<_AttributeValue> get copyWith => __$AttributeValueCopyWithImpl<_AttributeValue>(this, _$identity);



@override
bool operator ==(Object other) {
  return identical(this, other) || (other.runtimeType == runtimeType&&other is _AttributeValue&&(identical(other.value, value) || other.value == value)&&(identical(other.alt, alt) || other.alt == alt));
}


@override
int get hashCode => Object.hash(runtimeType,value,alt);

@override
String toString() {
  return 'AttributeValue(value: $value, alt: $alt)';
}


}

/// @nodoc
abstract mixin class _$AttributeValueCopyWith<$Res> implements $AttributeValueCopyWith<$Res> {
  factory _$AttributeValueCopyWith(_AttributeValue value, $Res Function(_AttributeValue) _then) = __$AttributeValueCopyWithImpl;
@override @useResult
$Res call({
 String? value, String? alt
});




}
/// @nodoc
class __$AttributeValueCopyWithImpl<$Res>
    implements _$AttributeValueCopyWith<$Res> {
  __$AttributeValueCopyWithImpl(this._self, this._then);

  final _AttributeValue _self;
  final $Res Function(_AttributeValue) _then;

/// Create a copy of AttributeValue
/// with the given fields replaced by the non-null parameter values.
@override @pragma('vm:prefer-inline') $Res call({Object? value = freezed,Object? alt = freezed,}) {
  return _then(_AttributeValue(
value: freezed == value ? _self.value : value // ignore: cast_nullable_to_non_nullable
as String?,alt: freezed == alt ? _self.alt : alt // ignore: cast_nullable_to_non_nullable
as String?,
  ));
}


}

// dart format on
